package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"log"
	"net"
	"sync"
	"time"
)

const (
  OSCClient_PacketsBuffer = 512
  OSCClient_UDPBuffer = 1024
  OSCClient_RetryTimeout = 5 * time.Second
)

type IOSCOutputPacket interface {
	Data() []byte
}

type OSCOutputStatus int32

const (
	OSCOutputStatus_OK 		OSCOutputStatus = 1
	OSCOutputStatus_Break OSCOutputStatus = 2
	OSCOutputStatus_Error OSCOutputStatus = 3
)

type OSCOutputResponsePacket struct {
  Path     string
  Index    int32
  Position int32
  Status   OSCOutputStatus
}

func (p *OSCOutputResponsePacket) Data() []byte {
	var buffer bytes.Buffer

	buffer.WriteString(p.Path)
	for (buffer.Len() % 4) != 0 {
		buffer.WriteByte(0)
	}

	buffer.Write([]byte{',', 'i', 'i', 'i', 0, 0, 0, 0})
	binary.Write(&buffer, binary.BigEndian, p.Status)
	binary.Write(&buffer, binary.BigEndian, p.Index)
	binary.Write(&buffer, binary.BigEndian, p.Position)

	return buffer.Bytes()
}

type OSCOutputPositionPacket struct {
	Path      string
	Positions [6]float32
}

func (p *OSCOutputPositionPacket) Data() []byte {
	var buffer bytes.Buffer

	buffer.WriteString(p.Path)
	for (buffer.Len() % 4) != 0 {
		buffer.WriteByte(0)
	}

	buffer.Write([]byte{',', 'f', 'f', 'f', 'f', 'f', 'f', 0})

	for _, position := range p.Positions {
		binary.Write(&buffer, binary.BigEndian, position)
	}

	return buffer.Bytes()
}

type OSCClient struct {
	addr *net.UDPAddr
  
  conn    		*net.UDPConn
  connMux 		sync.Mutex
  isConnected bool

  packets chan IOSCOutputPacket

  isShutdown bool
  wg sync.WaitGroup
}	

func NewOSCClient(address string) (*OSCClient, error) {
	addr, err := net.ResolveUDPAddr("udp", address)
	if err != nil {
		return nil, fmt.Errorf("OSCClient client failed to resolve UDP address: %w", err)
	}

	osc := &OSCClient{
		addr: addr,
		packets: make(chan IOSCOutputPacket, OSCClient_PacketsBuffer),
	}
	
	osc.wg.Add(1)
	go osc.processSendQueue()
	
	return osc, nil
}

func (osc *OSCClient) connect() error {
	for {
		osc.connMux.Lock()
		if osc.isConnected {
			osc.connMux.Unlock()
			return nil
		}

		var err error
		if osc.conn, err = net.DialUDP("udp", nil, osc.addr); err != nil {
			osc.connMux.Unlock()
			log.Printf("[OSCClient ERROR] Failed to reconnect: %v. Retrying in %.6f seconds...\n", err, OSCClient_RetryTimeout.Seconds())
			time.Sleep(OSCClient_RetryTimeout)
			continue
		}

		osc.isConnected = true
		osc.connMux.Unlock()
		log.Printf("[OSCClient INFO] Connected successfully.")
	}
}

func (osc *OSCClient) processSendQueue() {
	defer osc.wg.Done()
	for packet := range osc.packets {
		if err := osc.connect(); err != nil {
			log.Printf("[OSCClient ERROR] Failed to connect: %v\n", err)
			continue
		}

		osc.connMux.Lock()
		if _, err := osc.conn.Write(packet.Data()); err != nil {
			log.Printf("[OSCClient ERROR] Failed to send data: %v\n", err)
			osc.conn.Close()
			osc.isConnected = false
		}
		osc.connMux.Unlock()
	}
}

func (osc *OSCClient) Send(packet IOSCOutputPacket) {
	osc.packets <- packet
}

func (osc *OSCClient) Shutdown() {
  osc.isShutdown = true
  close(osc.packets)
  osc.wg.Wait()
  log.Printf("[OSCClient INFO] Client shutdown successfully\n")
}
