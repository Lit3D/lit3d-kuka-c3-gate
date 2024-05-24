package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"log"
	"net"
	"sync"
)

const (
  OSCServer_PacketsBuffer = 512
  OSCServer_UDPBuffer = 1024
)

var allIPv4 = net.ParseIP("0.0.0.0")

type OSCInputPacketType uint8

const (
  OSCInputPacketType_Command OSCInputPacketType = 1 // ,ii
  OSCInputPacketType_Coords  OSCInputPacketType = 2 // ,ffffff
)

type OSCCommandInputPacket struct {
  Path     string
  Index    int32
  Position int32
}

func (osc *OSCCommandInputPacket) Parse(data []byte) error {
  if len(data) < 8 {
    return fmt.Errorf("Invalid data length for ,ii type")
  }

  reader := bytes.NewReader(data)
  
  if err := binary.Read(reader, binary.BigEndian, &osc.Position); err != nil {
    return fmt.Errorf("Index parse error: %w", err)
  }

  if err := binary.Read(reader, binary.BigEndian, &osc.Index); err != nil {
    return fmt.Errorf("Position parse error: %w", err)
  }

  return nil
}

type OSCCoordsInputPacket struct {
  Path     string
  X float32
  Y float32
  Z float32
  A float32
  B float32
  C float32
}

func (osc *OSCCoordsInputPacket) Parse(data []byte) error {
  if len(data) < 8 {
    return fmt.Errorf("Invalid data length for ,ffffff type")
  }

  reader := bytes.NewReader(data)

  if err := binary.Read(reader, binary.BigEndian, &osc.X); err != nil {
    return fmt.Errorf("X coord parse error: %w", err)
  }

  if err := binary.Read(reader, binary.BigEndian, &osc.Y); err != nil {
    return fmt.Errorf("Y coord parse error: %w", err)
  }

  if err := binary.Read(reader, binary.BigEndian, &osc.Z); err != nil {
    return fmt.Errorf("Z coord parse error: %w", err)
  }

  if err := binary.Read(reader, binary.BigEndian, &osc.A); err != nil {
    return fmt.Errorf("A coord parse error: %w", err)
  }

  if err := binary.Read(reader, binary.BigEndian, &osc.B); err != nil {
    return fmt.Errorf("B coord parse error: %w", err)
  }

  if err := binary.Read(reader, binary.BigEndian, &osc.C); err != nil {
    return fmt.Errorf("C coord parse error: %w", err)
  }

  return nil
}


func ParseOSCInputPacket(packet []byte) (OSCInputPacketType, string, []byte, error) {
  log.Printf("%+v\n", packet)

  packetLength := len(packet)
  if packetLength < 8 {
    return 0, "", nil, fmt.Errorf("Invalid packet length of %d bytes", packetLength)
  }

  // Find the end of the address pattern (aligned to 4 bytes boundary)
  addrEnd := bytes.IndexByte(packet, 0)
  if addrEnd == -1 {
    return 0, "", nil, fmt.Errorf("Invalid address pattern")
  }

  // Extract the ptah string
  path := string(packet[:addrEnd])

  // Find the start of type tags (aligned to 4 bytes boundary)
  typeTagStart := addrEnd + 4 - (addrEnd % 4)
  if typeTagStart >= len(packet) || packet[typeTagStart] != ',' {
    return 0, "", nil, fmt.Errorf("Invalid type tag start")
  }

  // Find the end of the type tags (aligned to 4 bytes boundary)
  typeTagEnd := bytes.IndexByte(packet[typeTagStart:], 0)
  if typeTagEnd == -1 {
    return 0, "", nil, fmt.Errorf("Invalid type tags")
  }

  typeTags := string(packet[typeTagStart:typeTagStart + typeTagEnd])
  var dataType OSCInputPacketType
  switch typeTags {
  case ",ii":
    dataType = OSCInputPacketType_Command
  case ",ffffff":
    dataType = OSCInputPacketType_Coords
  default:
    return 0, "", nil, fmt.Errorf("Unsupported type tags")
  }

  // Calculate the start of the argument data (aligned to 4 bytes boundary)
  argDataStart := typeTagStart + typeTagEnd + (4 - ((typeTagStart + typeTagEnd) % 4))
  if argDataStart > len(packet) {
    return 0, "", nil, fmt.Errorf("No argument data")
  }

  data := packet[argDataStart:]

  return dataType, path, data, nil
}

type OSCServer struct {
  addr    net.UDPAddr
  conn    *net.UDPConn
  
  packets   chan []byte
  closeOnce sync.Once
  
  commandsSubscribers    map[string][]chan *OSCCommandInputPacket
  commandsSubscribersMux sync.RWMutex

  coordsSubscribers    map[string][]chan *OSCCoordsInputPacket
  coordsSubscribersMux sync.RWMutex

  wg sync.WaitGroup

  isShutdown bool
}

func NewOSCServer(port PortValue) *OSCServer {
  return &OSCServer{
    addr: net.UDPAddr{
      Port: int(port),
      IP:   allIPv4,
    },
    conn:        nil,
    packets:     make(chan []byte, OSCServer_PacketsBuffer),
    commandsSubscribers: make(map[string][]chan *OSCCommandInputPacket, OSCServer_PacketsBuffer),
    coordsSubscribers: make(map[string][]chan *OSCCoordsInputPacket, OSCServer_PacketsBuffer),
  }
}

func (osc *OSCServer) closePackets() {
  osc.closeOnce.Do(func() {
    close(osc.packets)
  })
}

func (osc *OSCServer) closeSubscribers() {
  osc.commandsSubscribersMux.Lock()
  defer osc.commandsSubscribersMux.Unlock()
  for key, subscribers := range osc.commandsSubscribers {
    for _, subscriber := range subscribers {
      close(subscriber)
    }
    osc.commandsSubscribers[key] = nil
  }

  osc.coordsSubscribersMux.Lock()
  defer osc.coordsSubscribersMux.Unlock()
  for key, subscribers := range osc.coordsSubscribers {
    for _, subscriber := range subscribers {
      close(subscriber)
    }
    osc.coordsSubscribers[key] = nil
  }
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
    packetType, path, data, err := ParseOSCInputPacket(packet)
    if err != nil {
      log.Printf("[OSCServer ERROR] Parse packet error: %v\n", err)
      continue
    }

    switch packetType {
      case OSCInputPacketType_Command:
        oscPacket := &OSCCommandInputPacket{Path: path}
        if err := oscPacket.Parse(data); err != nil {
          log.Printf("[OSCServer ERROR] Command packet parse error: %v\n", err)
          continue
        }
        osc.commandsSubscribersMux.RLock()
        subscribers, exists := osc.commandsSubscribers[oscPacket.Path]
        if !exists {
          osc.commandsSubscribersMux.RUnlock()
          continue
        }

        for _, subscriber := range subscribers {
          select {
            case subscriber <- oscPacket:
            default:
              log.Printf("[OSCServer WARNING] Subscribers channel is full, discarding packet\n")
          }
        }
        osc.commandsSubscribersMux.RUnlock()

      case OSCInputPacketType_Coords:
        oscPacket := &OSCCoordsInputPacket{Path: path}
        if err := oscPacket.Parse(data); err != nil {
          log.Printf("[OSCServer ERROR] Command packet parse error: %v\n", err)
          continue
        }

        osc.coordsSubscribersMux.RLock()
        subscribers, exists := osc.coordsSubscribers[oscPacket.Path]
        if !exists {
          osc.coordsSubscribersMux.RUnlock()
          continue
        }

        for _, subscriber := range subscribers {
          select {
            case subscriber <- oscPacket:
            default:
              log.Printf("[OSCServer WARNING] Subscribers channel is full, discarding packet\n")
          }
        }
        osc.coordsSubscribersMux.RUnlock()

    }
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
  osc.closeSubscribers()
  osc.wg.Wait()
  log.Printf("[OSCServer INFO] Server shutdown successfully\n")
}

func (osc *OSCServer) CommandsSubscribe(key string) chan *OSCCommandInputPacket {
  osc.commandsSubscribersMux.Lock()
  defer osc.commandsSubscribersMux.Unlock()

  subscriber := make(chan *OSCCommandInputPacket, OSCServer_PacketsBuffer)

  if _, exists := osc.commandsSubscribers[key]; !exists {
    osc.commandsSubscribers[key] = []chan *OSCCommandInputPacket{}
  }

  osc.commandsSubscribers[key] = append(osc.commandsSubscribers[key], subscriber)
  
  return subscriber
}

func (osc *OSCServer) CoordsSubscribe(key string) chan *OSCCoordsInputPacket {
  osc.coordsSubscribersMux.Lock()
  defer osc.coordsSubscribersMux.Unlock()

  subscriber := make(chan *OSCCoordsInputPacket, OSCServer_PacketsBuffer)

  if _, exists := osc.coordsSubscribers[key]; !exists {
    osc.coordsSubscribers[key] = []chan *OSCCoordsInputPacket{}
  }

  osc.coordsSubscribers[key] = append( osc.coordsSubscribers[key], subscriber)
  
  return subscriber
}
