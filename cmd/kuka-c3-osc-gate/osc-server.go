package main

import (
  "log"
  "net"
  "sync"
)

const (
  OSCServer_PacketsBuffer = 512
  OSCServer_UDPBuffer = 1024
)

type OSCListener interface {
   OSCPacket(oscPacket *OSCPacket)
}

type OSCServer struct {
  addr net.UDPAddr
  conn *net.UDPConn
  
  packets   chan []byte
  closeOnce sync.Once

  subscribers    []OSCListener
  subscribersMux sync.RWMutex

  wg sync.WaitGroup

  isShutdown bool
  debugFlag  bool
}

func NewOSCServer(addr net.UDPAddr) *OSCServer {
  return &OSCServer{
    addr: addr,
    conn: nil,
    
    packets:     make(chan []byte, OSCServer_PacketsBuffer),
    subscribers: make([]OSCListener, 0),
  }
}

func (osc *OSCServer) closePackets() {
  osc.closeOnce.Do(func() {
    close(osc.packets)
  })
}

func (osc *OSCServer) serve() {
  defer osc.wg.Done()

  buffer := make([]byte, OSCServer_UDPBuffer)
  for {
    n, _, err := osc.conn.ReadFromUDP(buffer)
    if err != nil {
      if netErr, ok := err.(net.Error); ok && netErr.Temporary() {
        continue
      }
      if osc.isShutdown == false {
        log.Printf("[OSCServer ERROR] Error reading from UDP: %v\n", err)
      }
      osc.closePackets()
      return
    }

    packet := make([]byte, n)
    copy(packet, buffer[:n])
    select {
      case osc.packets <- packet:
      default:
        log.Printf("[OSCServer WARNING] Packets channel is full, discarding packet\n")
    }
  }
}

func (osc *OSCServer) distributePackets() {
  defer osc.wg.Done()
  for packet := range osc.packets {
    oscPacket := NewOSCPacket()
    if err := oscPacket.Parse(packet); err != nil {
      log.Printf("[OSCServer ERROR] Packets parse error %v\n", err)
      continue
    }
    osc.subscribersMux.RLock()
    for _, listener := range osc.subscribers {
      listener.OSCPacket(oscPacket)
    }
    osc.subscribersMux.RUnlock()
  }
}

func (osc *OSCServer) ListenAndServe() error {
  conn, err := net.ListenUDP("udp", &osc.addr)
  if err != nil {
    return err
  }
  osc.conn = conn

  osc.wg.Add(1)
  go osc.serve()

  osc.wg.Add(1)
  go osc.distributePackets()

  log.Printf("[OSCServer INFO] Server start successfully at %s\n", osc.addr.String())
  return nil
}

func (osc *OSCServer) Shutdown() {
  osc.isShutdown = true
  osc.conn.Close()
  osc.closePackets()
  osc.wg.Wait()
  log.Printf("[OSCServer INFO] Server shutdown successfully\n")
}

func (osc *OSCServer) UnSubscribeAll() {
  osc.subscribersMux.Lock()
  osc.subscribers = make([]OSCListener, 0)
  osc.subscribersMux.Unlock()
}

func (osc *OSCServer) UnSubscribe(listener OSCListener) {
  for i, other := range osc.subscribers {
    if other == listener {
      osc.subscribersMux.Lock()
      osc.subscribers = append(osc.subscribers[:i], osc.subscribers[i+1:]...)
      osc.subscribersMux.Unlock()
      return
    }
  }
}

func (osc *OSCServer) Subscribe(listener OSCListener) {
  osc.UnSubscribe(listener)
  osc.subscribersMux.Lock()
  osc.subscribers = append(osc.subscribers, listener)
  osc.subscribersMux.Unlock()
}
