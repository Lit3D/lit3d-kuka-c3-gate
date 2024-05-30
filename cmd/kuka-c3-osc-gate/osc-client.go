package main

import (
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

type OSCClient struct {
  addr *net.UDPAddr
  
  conn        *net.UDPConn
  connMux     sync.Mutex
  isConnected bool

  packets chan []byte

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
    packets: make(chan []byte, OSCClient_PacketsBuffer),
  }
  
  osc.wg.Add(1)
  go osc.processPackets()
  
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

func (osc *OSCClient) processPackets() {
  defer osc.wg.Done()
  for packet := range osc.packets {
    if err := osc.connect(); err != nil {
      log.Printf("[OSCClient ERROR] Failed to connect: %v\n", err)
      continue
    }

    osc.connMux.Lock()
    if _, err := osc.conn.Write(packet); err != nil {
      log.Printf("[OSCClient ERROR] Failed to send data: %v\n", err)
      osc.conn.Close()
      osc.isConnected = false
    }
    osc.connMux.Unlock()
  }
}

func (osc *OSCClient) Send(packet *OSCPacket) error {
  data, err := packet.Bytes()
  if err != nil {
    return fmt.Errorf("OSCClient packet bytes error: %w", err)
  }
  osc.packets <- data
  return nil
}

func (osc *OSCClient) Shutdown() {
  osc.isShutdown = true
  close(osc.packets)
  osc.wg.Wait()
  log.Printf("[OSCClient INFO] Client shutdown successfully\n")
}
