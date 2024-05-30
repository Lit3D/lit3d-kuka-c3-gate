package main

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"
)

const (
  Bot_PacketsBuffer = 512
  Bot_MessagesBuffer = 512

  Bot_C3_Request_Timeout = 3 * time.Second

  Bot_Position_Tolerance = 0.0100
  Bot_Position_ReadySteps = 5

  Bot_Move_Timeout = 30 * time.Second
)

type Bot struct {
	Name    string `json:"name"`
  Address string `json:"address"`

  AxisPath     string `json:"axisPath"`
  CoordsPath   string `json:"coordsPath"`
  PositionPath string `json:"positionPath"`

  ResponseAddress  *string `json:"responseAddress"`
  
  ResponseAxes     *string `json:"responseAxes"`
  ResponseCoords   *string `json:"responseCoords"`
  ResponsePosition *string `json:"responsPosition"`

  Positions []*Position `json:"positions"`

  OSCInput chan *OSCPacket

  tagId    uint16
  tagIdMux sync.RWMutex
  
  c3Client  *C3Client
  oscClient *OSCClient

  isMovement bool
  isMovementMux sync.RWMutex

  AXIS_ACT    *Position
  POS_ACT     *Position
  COM_ACTION  C3VariableComActionValues
  COM_ROUNDM  C3VariableComRoundmValues
  POSITION    *Position
  positionMux sync.RWMutex

  OFFSET    *Position
  offsetMux sync.RWMutex

  PROXY_TYPE     string
  PROXY_VERSION  string
  PROXY_HOSTNAME string
  PROXY_ADDRESS  string
  PROXY_PORT     string
  proxyMux       sync.RWMutex

  isShutdown bool
  wg sync.WaitGroup
}

func NewBot() (*Bot, error) {
  return &Bot{
    tagId:      1,
    OSCInput:   make(chan *OSCPacket, Bot_PacketsBuffer),
    AXIS_ACT:   NewPosition(0xFFFF, PositionType_E6AXIS),
    POS_ACT:    NewPosition(0xFFFF, PositionType_E6POS),
    POSITION:   NewPosition(0xFFFF, PositionType_E6POS),
    OFFSET:     NewPosition(0xFFFF, PositionType_E6POS),
    COM_ACTION: C3Variable_COM_ACTION_EMPTY,
    COM_ROUNDM: C3Variable_COM_ROUNDM_NONE,
  }, nil
}

func (bot *Bot) MarshalJSON() ([]byte, error) {
  return json.Marshal(bot)
}

func (bot *Bot) UnmarshalJSON(data []byte) error {
  bot.tagId = 1
  
  bot.OSCInput = make(chan *OSCPacket, Bot_PacketsBuffer)
  bot.AXIS_ACT = NewPosition(0xFFFF, PositionType_E6AXIS)
  bot.POS_ACT  = NewPosition(0xFFFF, PositionType_E6POS)
  bot.POSITION = NewPosition(0xFFFF, PositionType_E6POS)
  bot.OFFSET   = NewPosition(0xFFFF, PositionType_E6POS)

  bot.COM_ACTION = C3Variable_COM_ACTION_EMPTY
  bot.COM_ROUNDM = C3Variable_COM_ROUNDM_NONE

  return json.Unmarshal(data, bot)
}

func (bot *Bot) Up() (err error) {
  if bot.c3Client, err = NewC3Client(bot.Address); err != nil {
    return fmt.Errorf("C3Client creation error: %w", err)
  }

  if bot.ResponseAddress != nil {
    if bot.oscClient, err = NewOSCClient(*bot.ResponseAddress); err != nil {
      return fmt.Errorf("OSCClient creation error: %w", err)
    }
  } else {
    bot.oscClient = nil
  }

  if err := bot.UpdateProxyInfo(); err != nil {
    return fmt.Errorf("Proxy get info error: %w", err)
  }

  if err := bot.UpdatePosition(); err != nil {
    return fmt.Errorf("Position update error: %w", err)
  }

  if err := bot.UpdateOffset(); err != nil {
    return fmt.Errorf("Offset update error: %w", err)
  }

  bot.LogBot()
  bot.isShutdown = false

  bot.wg.Add(1)
  go bot.processOSCPackets()

  bot.wg.Add(1)
  go bot.processUpdatePosition()

  return nil
}

func (bot *Bot) Down() error {
  bot.isShutdown = true
  close(bot.OSCInput)
  bot.oscClient.Shutdown()
  bot.c3Client.Shutdown()
  bot.wg.Wait()
  log.Printf("[Bot INFO] Bot shutdown successfully\n")
  return nil
}

func (bot *Bot) LogBot() {
  bot.proxyMux.RLock()
  defer bot.proxyMux.RUnlock()
  bot.positionMux.RLock()
  defer bot.positionMux.RUnlock()
  bot.isMovementMux.RLock()
  defer bot.isMovementMux.RUnlock()
  bot.tagIdMux.RLock()
  defer bot.tagIdMux.RUnlock()
  bot.offsetMux.RLock()
  defer bot.offsetMux.RUnlock()
  log.Printf(
    "==========> Bot: %s[%s]\n" +
    "AxisPath: %s CoordsPath: %s PositionPath: %s\n" + 
    "ResponseAddress: %v ResponseAxes: %v ResponseCoords: %v ResponsePosition: %v\n" +
    "ProxyType: %s ProxyVersion: %s ProxyHost: %s ProxyAddress: %s ProxyPort: %s\n" +
    "AXIS_ACT: %s\n" +
    "POS_ACT: %s\n" +
    "OFFSET: %s\n" +
    "POSITION: %s\n" +
    "COM_ACTION: %s COM_ROUNDM: %s isMovement: %t tagId: %d",
    bot.Name, bot.Address,
    bot.AxisPath, bot.CoordsPath, bot.PositionPath,
    bot.ResponseAddress, bot.ResponseAxes, bot.ResponseCoords, bot.ResponsePosition, 
    bot.PROXY_TYPE, bot.PROXY_VERSION, bot.PROXY_HOSTNAME, bot.PROXY_ADDRESS, bot.PROXY_PORT,
    bot.AXIS_ACT.ValueFull(),
    bot.POS_ACT.ValueFull(),
    bot.OFFSET.ValueFull(),
    bot.POSITION.ValueFull(),
    bot.COM_ACTION, bot.COM_ROUNDM, bot.isMovement, bot.tagId,
  )
}

func (bot *Bot) nextTagId() uint16 {
  bot.tagIdMux.Lock()
  defer bot.tagIdMux.Unlock()
  
  currentTagId := bot.tagId
  currentTagId += 1
  if (currentTagId >= 65535) {
    currentTagId = 1
  }
  bot.tagId = currentTagId

  return currentTagId
}

func (bot *Bot) Move(p *Position) (bool, error) {
  bot.isMovementMux.Lock()
  defer bot.isMovementMux.Unlock()

  if bot.isMovement == true {
    return false, fmt.Errorf("Bot already movement")
  }

  requestPositionVariable := make(map[C3VariableType]*string)
  
  positionType  := p.Type()
  positionValue := p.Value()

  if positionType == PositionType_E6AXIS {
    requestPositionVariable[C3Variable_COM_E6AXIS] = &positionValue
  } else if positionType == PositionType_E6POS {
    requestPositionVariable[C3Variable_COM_E6POS] = &positionValue
  } else {
    return false, fmt.Errorf("Incorrect Move position type of %d", positionType)
  }

  comRoundmValue := string(C3Variable_COM_ROUNDM_NONE)
  requestPositionVariable[C3Variable_COM_ROUNDM] = &comRoundmValue

  positionMessage, err := NewC3Message(bot.nextTagId(), requestPositionVariable)
  if err != nil {
    return false, fmt.Errorf("Move new Position message error: %w", err)
  }

  requestComActionVariable := make(map[C3VariableType]*string)
  comActionValue := string(C3Variable_COM_ROUNDM_NONE)
  requestComActionVariable[C3Variable_COM_ACTION] = &comActionValue

  comActionMessage, err := NewC3Message(bot.nextTagId(), requestComActionVariable)
  if err != nil {
    return false, fmt.Errorf("Move new COM_ACTION message error: %w", err)
  }

  resultPositionChan, err := bot.c3Client.Request(positionMessage)
  if err != nil {
    return false, fmt.Errorf("Move Possition message request error: %w", err)
  }

  resultComActionChan, err := bot.c3Client.Request(comActionMessage)
  if err != nil {
    return false, fmt.Errorf("Move COM_ACTION message request error: %w", err)
  }

  select {
    case positionMessage = <-resultPositionChan:
      err = positionMessage.Error()
      if err != nil {
        return false, fmt.Errorf("Move Possition message result error: %w", err)
      }
    case <-time.After(Bot_C3_Request_Timeout):
      return false, fmt.Errorf("Move Possition message request timeout")
  }

  select {
    case comActionMessage = <-resultComActionChan:
      err = comActionMessage.Error()
      if err != nil {
        return false, fmt.Errorf("Move COM_ACTION message result error: %w", err)
      }
    case <-time.After(Bot_C3_Request_Timeout):
      return false, fmt.Errorf("Move COM_ACTION message request timeout")
  }

  bot.isMovement = true
  log.Printf("[Bot INFO] Move bot to position %s\n", p.Value())
  
  var readyFlag uint8 = 0
  timeout := time.After(Bot_Move_Timeout)
  
  loop: for {
    select {
      case <-timeout:
        return true, fmt.Errorf("Move timeout break")
      
      default:
        // if bot.UpdatePosition(); err != nil {
        //   log.Printf("[Bot ERROR] Move Get position error %v\n", err)
        // }

        bot.positionMux.RLock()
        if bot.POSITION.Equal(p, Bot_Position_Tolerance) {
          readyFlag++
        }
        bot.positionMux.RUnlock()

        if readyFlag >= Bot_Position_ReadySteps {
          break loop
        }
    }
  }

  log.Printf("[Bot INFO] Move ready position %s\n", p.Value())
  bot.isMovement = false
  
  return false, nil
}

func (bot *Bot) UpdatePosition() error {
  requestVariable := make(map[C3VariableType]*string)
  
  requestVariable[C3Variable_AXIS_ACT] = nil
  requestVariable[C3Variable_POS_ACT] = nil
  requestVariable[C3Variable_COM_ACTION] = nil
  requestVariable[C3Variable_COM_ROUNDM] = nil

  message, err := NewC3Message(bot.nextTagId(), requestVariable)
  if err != nil {
    return fmt.Errorf("Get Position new message error: %w", err)
  }

  resultChan, err := bot.c3Client.Request(message)
  if err != nil {
    return fmt.Errorf("Get Position message request error: %w", err)
  }

  select {
    case message = <-resultChan:
      err = message.Error()
      if err != nil {
        return fmt.Errorf("Get Position message result error: %w", err)
      }
    case <-time.After(Bot_C3_Request_Timeout):
      return fmt.Errorf("Get Position message request timeout")
  }

  var AXIS_ACT *Position = NewPosition(0xFFFF, PositionType_NIL)
  var POS_ACT *Position = NewPosition(0xFFFF, PositionType_NIL)
  var COM_ACTION C3VariableComActionValues = C3Variable_COM_ACTION_EMPTY
  var COM_ROUNDM C3VariableComRoundmValues = C3Variable_COM_ROUNDM_NONE

  for _, variable := range message.Variables() {
    if variable.ErrorCode != C3Message_Error_Success {
      return fmt.Errorf("Get Position message result variable %s error: %s", variable.Name, C3ErrorString[variable.ErrorCode])
    }

    switch variable.Name {
      case C3Variable_AXIS_ACT:
        if err := AXIS_ACT.Parse(variable.Value); err != nil {
          return fmt.Errorf("Get Position variable %s error: %w", variable.Name, err)
        }

      case C3Variable_POS_ACT:
        if err := POS_ACT.Parse(variable.Value); err != nil {
          return fmt.Errorf("Get Position variable %s error: %w", variable.Name, err)
        }

      case C3Variable_COM_ACTION:
        COM_ACTION = C3VariableComActionValues(variable.Value)

      case C3Variable_COM_ROUNDM:
        COM_ROUNDM = C3VariableComRoundmValues(variable.Value)
    }
  }

  bot.positionMux.Lock()
  bot.AXIS_ACT = AXIS_ACT
  bot.POS_ACT = POS_ACT
  bot.COM_ACTION = COM_ACTION
  bot.COM_ROUNDM = COM_ROUNDM
  bot.offsetMux.RUnlock()
  bot.POSITION = POS_ACT.WithOffset(bot.OFFSET)
  bot.offsetMux.RUnlock()
  bot.positionMux.Unlock()

  return nil
}

func (bot *Bot) UpdateOffset() error {
  bot.positionMux.RLock()
  defer bot.positionMux.RUnlock()
  bot.offsetMux.Lock()
  defer bot.offsetMux.Unlock()
  bot.OFFSET = bot.POS_ACT.Clone()
  return nil
}

func (bot *Bot) processUpdatePosition() {
  defer bot.wg.Done()
  for {
    if bot.isShutdown == true {
      return
    }
    if err := bot.UpdatePosition(); err != nil {
      if bot.isShutdown == true {
        return
      }
      log.Printf("[Bot ERROR] UpdatePosition Get position error %v\n", err)
      continue
    }

    if err := bot.oscResponseCurrentCoords(); err != nil {
      log.Printf("[Bot ERROR] Response current coords error %v\n", err)
    }
  }
}

func (bot *Bot) UpdateProxyInfo() error {
  requestVariable := make(map[C3VariableType]*string)
  
  requestVariable[C3Variable_PROXY_TYPE]     = nil
  requestVariable[C3Variable_PROXY_VERSION]  = nil
  requestVariable[C3Variable_PROXY_HOSTNAME] = nil
  requestVariable[C3Variable_PROXY_ADDRESS]  = nil
  requestVariable[C3Variable_PROXY_PORT]     = nil

  message, err := NewC3Message(bot.nextTagId(), requestVariable)
  if err != nil {
    return fmt.Errorf("Get Proxy info new message error: %w", err)
  }

  resultChan, err := bot.c3Client.Request(message)
  if err != nil {
    return fmt.Errorf("Get Proxy info message request error: %w", err)
  }

  resultMessage := <- resultChan
  err = resultMessage.Error()
  if err != nil {
    return fmt.Errorf("Get Proxy info message result error: %w", err)
  }

  bot.proxyMux.Lock()
  for _, variable := range resultMessage.Variables() {
    if variable.ErrorCode != C3Message_Error_Success {
      return fmt.Errorf("Get Proxy info result variable %s error: %s", variable.Name, C3ErrorString[variable.ErrorCode])
    }

    switch variable.Name {
      case C3Variable_PROXY_TYPE:
        bot.PROXY_TYPE = variable.Value

      case C3Variable_PROXY_VERSION:
        bot.PROXY_VERSION = variable.Value

      case C3Variable_PROXY_HOSTNAME:
        bot.PROXY_HOSTNAME = variable.Value

      case C3Variable_PROXY_ADDRESS:
        bot.PROXY_ADDRESS = variable.Value

      case C3Variable_PROXY_PORT:
        bot.PROXY_PORT = variable.Value
    }
  }
  bot.proxyMux.Unlock()

  return nil
}

func (bot *Bot) processOSCPackets() {
  defer bot.wg.Done()

  for packet := range bot.OSCInput {
    switch packet.Path {
      case bot.AxisPath:
        bot.processOSCAxis(packet)
      case bot.CoordsPath:
        bot.processOSCCoords(packet)
      case bot.PositionPath:
        bot.processOSCPosition(packet)
    }
  }
}

func (bot *Bot) processOSCAxis(oscPacket *OSCPacket) {
  values := oscPacket.Values()
  if len(values) != 6 {
    log.Printf("[Bot ERROR] Incorrect OSC values length of %+v\n", values)
    return
  }

  position := NewPosition(0xFFFF, PositionType_E6AXIS)
  for i, value := range values {
    switch value.(type) {
      case float32:
        position.Set(i, value.(float32))
      default:
        log.Printf("[Bot ERROR] OSC values[%d] is not of float32 value\n", i)
        return
    }
  }

  go func(position *Position) {
    if _, err := bot.Move(position); err != nil {
      log.Printf("[Bot ERROR] OSC Position %s move error: %v\n", position.Value(), err)
    }
  }(position)
}

func (bot *Bot) processOSCCoords(oscPacket *OSCPacket) {
  values := oscPacket.Values()
  if len(values) != 6 {
    log.Printf("[Bot ERROR] Incorrect OSC values length of %+v\n", values)
    return
  }

  position := NewPosition(0xFFFF, PositionType_E6POS)
  for i, value := range values {
    switch value.(type) {
      case float32:
        position.Set(i, value.(float32))
      default:
        log.Printf("[Bot ERROR] OSC values[%d] is not of float32 value\n", i)
        return
    }
  }

  go func(position *Position) {
    if _, err := bot.Move(position); err != nil {
      log.Printf("[Bot ERROR] OSC Position %s move error: %v\n", position.Value(), err)
    }
  }(position)
}

func (bot *Bot) processOSCPosition(oscPacket *OSCPacket) {
  values := oscPacket.Values()
  if len(values) != 2 {
    log.Printf("[Bot ERROR] Incorrect OSC Position values length of %+v\n", values)
    return
  }

  var position *Position = nil
  var positionId uint16 = uint16(values[0].(int32))
  var index int32 = values[1].(int32)

  for _, position = range bot.Positions {
    if position.Id() == positionId {
      break
    }
  }

  if position == nil {
    log.Printf("[Bot ERROR] Incorrect OSC Position id of %d\n", positionId)
    go func() {
      if err := bot.oscResponsePosition(OSCOutputStatus_Error, index, positionId); err != nil {
        log.Printf("[Bot ERROR] OSC Response error %v\n", err)
      }
    }()
    return
  }

  go func(index int32, position *Position) {
    if isBreak, err := bot.Move(position); err != nil {
      log.Printf("[Bot ERROR] OSC Position %d:%s move error: %v\n", position.Id(), position.Value(), err)
      status := OSCOutputStatus_Error
      if isBreak == true {
        status = OSCOutputStatus_Break
      }
      if err := bot.oscResponsePosition(status, index, position.Id()); err != nil {
        log.Printf("[Bot ERROR] OSC error move response error %v\n", err)
      }
    }
    if err := bot.oscResponsePosition(OSCOutputStatus_OK, index, position.Id()); err != nil {
      log.Printf("[Bot ERROR] OSC sucess move response error %v\n", err)
    }
  }(index, position)
}

func (bot *Bot) oscResponseAxis(position *Position) error {
  if bot.oscClient == nil {
    return nil
  }

  if bot.ResponseAxes == nil {
    return nil
  }

  oscPacker := NewOSCPacket()
  oscPacker.Path = *bot.ResponseAxes
  oscPacker.Append(position.A1)
  oscPacker.Append(position.A2)
  oscPacker.Append(position.A3)
  oscPacker.Append(position.A4)
  oscPacker.Append(position.A5)
  oscPacker.Append(position.A6)
  return bot.oscClient.Send(oscPacker)
}

func (bot *Bot) oscResponseCoords(position *Position) error {
  if bot.oscClient == nil {
    return nil
  }

  if bot.ResponseCoords == nil {
    return nil
  }

  oscPacker := NewOSCPacket()
  oscPacker.Path = *bot.ResponseCoords
  oscPacker.Append(position.X)
  oscPacker.Append(position.Y)
  oscPacker.Append(position.Z)
  oscPacker.Append(position.A)
  oscPacker.Append(position.B)
  oscPacker.Append(position.C)
  // oscPacker.Append(position.S)
  // oscPacker.Append(position.T)
  return bot.oscClient.Send(oscPacker)
}

func (bot *Bot) oscResponsePosition(status OSCOutputStatus, index int32, positionId uint16) error {
  if bot.oscClient == nil {
    return nil
  }

  if bot.ResponsePosition == nil {
    return nil
  }

  oscPacker := NewOSCPacket()
  oscPacker.Path = *bot.ResponsePosition
  oscPacker.Append(int32(status))
  oscPacker.Append(int32(index))
  oscPacker.Append(int32(positionId))
  return bot.oscClient.Send(oscPacker)
}

func (bot *Bot) oscResponseCurrentCoords() error {
  bot.positionMux.RLock()
  defer bot.positionMux.RUnlock()
  position := bot.POSITION.Clone()
  return bot.oscResponseCoords(position)
}
