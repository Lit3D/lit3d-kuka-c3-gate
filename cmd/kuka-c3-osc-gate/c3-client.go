package main

import (
	"encoding/binary"
	"fmt"
	"log"
	"net"
	"sync"
	"time"
)

const (
  C3Client_PacketsBuffer = 512
  C3Client_TCPBuffer = 2048
  C3Client_RetryTimeout = 5 * time.Second
)

type AsyncC3Message struct {
  Message    *C3Message
  ResultChan chan *C3Message
}

type C3Client struct {
  addr *net.TCPAddr

  conn        *net.TCPConn
  connMux     sync.Mutex
  isConnected bool

  messageStore    map[uint16]*AsyncC3Message
  messageStoreMux sync.Mutex

  requestPackets  chan []byte
  responsePackets chan []byte

  isShutdown bool

  closeOnce sync.Once
  wg sync.WaitGroup
}

func NewC3Client(address string) (*C3Client, error) {
  addr, err := net.ResolveTCPAddr("tcp4", address)
  if err != nil {
    return nil, fmt.Errorf("C3Client client failed to resolve TCP address: %w", err)
  }

  с3 := &C3Client{
    addr: addr,
    messageStore: make(map[uint16]*AsyncC3Message),
    requestPackets:  make(chan []byte, C3Client_PacketsBuffer),
    responsePackets: make(chan []byte, C3Client_PacketsBuffer),
  }

  с3.wg.Add(1)
  go с3.processRequest()

  с3.wg.Add(1)
  go с3.processResponse()

  с3.wg.Add(1)
  go с3.processMessage()

  return с3, nil
}

func (c3 *C3Client) setMessage(msg *C3Message) *AsyncC3Message {
  c3.messageStoreMux.Lock()
  defer c3.messageStoreMux.Unlock()
  
  asyncMessage := &AsyncC3Message{Message: msg, ResultChan: make(chan *C3Message)}
  c3.messageStore[msg.TagID(nil)] = asyncMessage
  return asyncMessage
}

func (c3 *C3Client) getMessage(tagID uint16) *AsyncC3Message {
  c3.messageStoreMux.Lock()
  defer c3.messageStoreMux.Unlock()
  
  if msg, ok := c3.messageStore[tagID]; ok {
    delete(c3.messageStore, tagID)
    return msg
  }

  return nil
} 

func (c3 *C3Client) connect() error {
  for {
    c3.connMux.Lock()
    if c3.isConnected {
      c3.connMux.Unlock()
      return nil
    }

    var err error
    if c3.conn, err = net.DialTCP("tcp", nil, c3.addr); err != nil {
      c3.connMux.Unlock()
      log.Printf("[C3Client ERROR] Failed to reconnect: %v. Retrying in %.6f seconds...\n", err, C3Client_RetryTimeout.Seconds())
      time.Sleep(C3Client_RetryTimeout)
      continue
    }

    c3.isConnected = true
    c3.connMux.Unlock()
    log.Printf("[C3Client INFO] Connected successfully to %s\n", c3.addr.String())
  }
}

func (c3 *C3Client) processRequest() {
  defer c3.wg.Done()
  for packet := range c3.requestPackets {
    if err := c3.connect(); err != nil {
      log.Printf("[C3Client ERROR] Failed to connect: %v\n", err)
      continue
    }

    c3.connMux.Lock()
    if _, err := c3.conn.Write(packet); err != nil {
      log.Printf("[C3Client ERROR] Failed to send data: %v\n", err)
      c3.conn.Close()
      c3.isConnected = false
    }
    c3.connMux.Unlock()
  }
}

func (c3 *C3Client) processResponse() {
  defer c3.wg.Done()
  for {
    if c3.isShutdown == true {
      return  
    }
    
    if err := c3.connect(); err != nil {
      log.Printf("[C3Client ERROR] Failed to connect: %v\n", err)
      continue
    }

    buffer := make([]byte, C3Client_TCPBuffer)
    for {
      n, err := c3.conn.Read(buffer)
      if err != nil {
        if netErr, ok := err.(net.Error); ok && netErr.Temporary() {
          continue
        }
        if c3.isShutdown == false {
          log.Printf("[C3Client ERROR] Failed to read response: %v\n", err)
        }
        c3.conn.Close()
        c3.isConnected = false
        break
      }
      packet := make([]byte, n)
      copy(packet, buffer[:n])
      select {
        case c3.responsePackets <- packet:
        default:
          log.Printf("[C3Client WARNING] Response packets channel is full, discarding packet\n")
      }
    }
  }
}

func (c3 *C3Client) processMessage() {
  defer c3.wg.Done()
  for packet := range c3.responsePackets {
    if len(packet) < 8 {
      log.Printf("[C3Client ERROR] Packet is too short: %v\n", packet)
      continue
    }

    tagID := binary.BigEndian.Uint16(packet[:2])
    asyncMessage := c3.getMessage(tagID)
    if asyncMessage == nil {
       log.Printf("[C3Client ERROR] Request packet TagId[%d] is not found\n", tagID)
       continue
    }
    
    if err := asyncMessage.Message.Response(packet); err != nil {
      log.Printf("[C3Client ERROR] Response packet parse error: %v\n", err)
      continue
    }

    asyncMessage.ResultChan <- asyncMessage.Message
    close(asyncMessage.ResultChan)
  }
}

func (c3 *C3Client) closePackets() {
  c3.closeOnce.Do(func() {
    close(c3.requestPackets)
    close(c3.responsePackets)
  })
}

func (c3 *C3Client) Request(message *C3Message) (chan *C3Message, error) {
  packet, err := message.Request()
  if err != nil {
    return nil, fmt.Errorf("C3Client client failed to get Request data: %w", err)
  }
  
  asyncMessage := c3.setMessage(message)
  c3.requestPackets <- packet
  return asyncMessage.ResultChan, nil
}

func (c3 *C3Client) Shutdown() {
  c3.isShutdown = true
  c3.conn.Close()
  c3.closePackets()
  c3.wg.Wait()
  log.Printf("[C3Client INFO] Client shutdown successfully\n")
}
