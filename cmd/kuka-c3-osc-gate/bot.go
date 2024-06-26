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
  Bot_Position_ReadySteps = 100

  Bot_Move_Timeout = 60 * time.Second
)

type Bot struct {
	Name    string `json:"name"`
  Address string `json:"address"`

  OSCRequestAxis     *string `json:"oscRequestAxisPath"`
  OSCRequestCoords   *string `json:"oscRequestCoordsPath"`
  OSCRequestPosition *string `json:"oscRequestPositionPath"`

  OSCResponseAddress  *string `json:"oscResponseAddress"`
  
  OSCResponseAxes     *string `json:"oscResponseAxes"`
  OSCResponseCoords   *string `json:"oscResponseCoords"`
  OSCResponsePosition *string `json:"oscResponsPosition"`

  MoveGroups []*MoveGroup `json:"moveGroups"`
  moveGroupsMux sync.RWMutex

  oscInput  chan *OSCPacket
  moveInput chan *MoveGroup

  tagId    uint16
  tagIdMux sync.RWMutex
  
  c3Client  *C3Client
  oscClient *OSCClient

  isMovement bool
  isMovementMux sync.RWMutex

  c3AXIS_ACT   *Position
  c3POS_ACT    *Position
  c3COM_ACTION C3VariableComActionValues
  c3COM_ROUNDM C3VariableComRoundmValues
  c3OFFSET     *Position
  c3POSITION   *Position
  positionMux sync.RWMutex

  c3PROXY_TYPE     string
  c3PROXY_VERSION  string
  c3PROXY_HOSTNAME string
  c3PROXY_ADDRESS  string
  c3PROXY_PORT     string
  proxyMux       sync.RWMutex

  isShutdown bool
  wg sync.WaitGroup

  currentMoveGroupId uint16
  currentMoveGroupIdMux sync.RWMutex
}

func NewBot() (*Bot, error) {
  return &Bot{
    tagId:        1,
    c3AXIS_ACT:   NewPosition(PositionType_E6AXIS),
    c3POS_ACT:    NewPosition(PositionType_E6POS),
    c3POSITION:   NewPosition(PositionType_E6POS),
    c3OFFSET:     NewPosition(PositionType_E6POS),
    c3COM_ACTION: C3Variable_COM_ACTION_EMPTY,
    c3COM_ROUNDM: C3Variable_COM_ROUNDM_NONE,
    MoveGroups:   make([]*MoveGroup, 0),
  }, nil
}

func (bot *Bot) MarshalJSON() ([]byte, error) {
  type Alias Bot
  return json.Marshal(&struct {
    *Alias
  }{
    Alias: (*Alias)(bot),
  })
}

func (bot *Bot) UnmarshalJSON(data []byte) error {
  type Alias Bot
  aux := &struct {
    *Alias
  }{
    Alias: (*Alias)(bot),
  }

  if err := json.Unmarshal(data, &aux); err != nil {
    return err
  }

  bot.tagId        = 1
  bot.c3AXIS_ACT   = NewPosition(PositionType_E6AXIS)
  bot.c3POS_ACT    = NewPosition(PositionType_E6POS)
  bot.c3POSITION   = NewPosition(PositionType_E6POS)
  bot.c3OFFSET     = NewPosition(PositionType_E6POS)
  bot.c3COM_ACTION = C3Variable_COM_ACTION_EMPTY
  bot.c3COM_ROUNDM = C3Variable_COM_ROUNDM_NONE
  return nil
}

func (bot *Bot) Up() (err error) {
  bot.oscInput = make(chan *OSCPacket, Bot_PacketsBuffer)
  bot.moveInput = make(chan *MoveGroup, Bot_PacketsBuffer)
  bot.isShutdown = false

  if bot.c3Client, err = NewC3Client(bot.Address); err != nil {
    return fmt.Errorf("Bot %s C3Client creation error: %w", bot.Name, err)
  }

  if bot.OSCResponseAddress != nil {
    if bot.oscClient, err = NewOSCClient(*bot.OSCResponseAddress); err != nil {
      return fmt.Errorf("Bot %s OSCClient creation error: %w", bot.Name, err)
    }
  } else {
    bot.oscClient = nil
  }

  if err := bot.UpdateProxyInfo(); err != nil {
    return fmt.Errorf("Bot %s Proxy get info error: %w", bot.Name, err)
  }

  if err := bot.UpdatePosition(); err != nil {
    return fmt.Errorf("Bot %s Position update error: %w", bot.Name, err)
  }

  HOME := NewPosition(PositionType_E6AXIS)
  bot.positionMux.RLock()
  HOME.SetValues([14]float32{0, -90, 90, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0})
  if bot.c3AXIS_ACT.Equal(HOME, Bot_Position_Tolerance) != true {
     return fmt.Errorf("Bot %s is not HOME position", bot.Name)
  }

  bot.positionMux.RUnlock()


  if err := bot.ResetOffsetAndPosition(); err != nil {
    return fmt.Errorf("Bot %s Offset and Position update error: %w", bot.Name, err)
  }

  bot.LogBot()

  bot.wg.Add(1)
  go bot.processOSCPackets()

  bot.wg.Add(1)
  go bot.processMoveGroup()

  bot.wg.Add(1)
  go bot.processUpdatePosition()

  return nil
}

func (bot *Bot) Shutdown() error {
  bot.isShutdown = true
  close(bot.oscInput)
  close(bot.moveInput)
  if bot.oscClient != nil {
    bot.oscClient.Shutdown()
  }
  bot.c3Client.Shutdown()
  bot.wg.Wait()
  log.Printf("[Bot %s INFO] Shutdown successfully\n", bot.Name)
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

  log.Printf(
    "==========> Bot: %s[%s]\n" +
    "AxisPath: %s CoordsPath: %s PositionPath: %s\n" + 
    "ResponseAddress: %s ResponseAxes: %s ResponseCoords: %s ResponsePosition: %s\n" +
    "ProxyType: %s ProxyVersion: %s ProxyHost: %s ProxyAddress: %s ProxyPort: %s\n" +
    "AXIS_ACT: %s\n" +
    "POS_ACT: %s\n" +
    "OFFSET: %s\n" +
    "POSITION: %s\n" +
    "COM_ACTION: %s COM_ROUNDM: %s isMovement: %t tagId: %d",
    bot.Name, bot.Address,
    nilStringToString(bot.OSCRequestAxis), nilStringToString(bot.OSCRequestCoords), nilStringToString(bot.OSCRequestPosition),
    nilStringToString(bot.OSCResponseAddress), nilStringToString(bot.OSCResponseAxes), nilStringToString(bot.OSCResponseCoords), nilStringToString(bot.OSCResponsePosition), 
    bot.c3PROXY_TYPE, bot.c3PROXY_VERSION, bot.c3PROXY_HOSTNAME, bot.c3PROXY_ADDRESS, bot.c3PROXY_PORT,
    bot.c3AXIS_ACT.ValueFull(),
    bot.c3POS_ACT.ValueFull(),
    bot.c3OFFSET.ValueFull(),
    bot.c3POSITION.ValueFull(),
    bot.c3COM_ACTION, bot.c3COM_ROUNDM, bot.isMovement, bot.tagId,
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

func (bot *Bot) SetSpeed(VEL_CP uint8, ACC_CP uint8) error {
  bot.isMovementMux.RLock()
  if bot.isMovement == true {
    bot.isMovementMux.RUnlock()
    return fmt.Errorf("Bot is movement")
  }
  bot.isMovementMux.RUnlock()

  requestSpeedVariable  := make(map[C3VariableType]*string)
  requestComActionVariable := make(map[C3VariableType]*string)

  VEL_CPValue := fmt.Sprintf("%d", VEL_CP)
  requestSpeedVariable[C3Variable_COM_VALUE1] = &VEL_CPValue

  ACC_CPValue := fmt.Sprintf("%d", ACC_CP)
  requestSpeedVariable[C3Variable_COM_VALUE3] = &ACC_CPValue

  comActionValue := string(C3Variable_COM_ACTION_VEL_AXIS)
  requestComActionVariable[C3Variable_COM_ACTION] = &comActionValue

  speedMessage, err := NewC3Message(bot.nextTagId(), requestSpeedVariable)
  if err != nil {
    return fmt.Errorf("Speed value message error: %w", err)
  }

  comActionMessage, err := NewC3Message(bot.nextTagId(), requestComActionVariable)
  if err != nil {
    return fmt.Errorf("Speed COM_ACTION message error: %w", err)
  }

  resultSpeedChan, err := bot.c3Client.Request(speedMessage)
  if err != nil {
    return fmt.Errorf("Speed value message request error: %w", err)
  }

  resultComActionChan, err := bot.c3Client.Request(comActionMessage)
  if err != nil {
    return fmt.Errorf("Speed COM_ACTION message request error: %w", err)
  }

  select {
    case speedMessage = <-resultSpeedChan:
      err = speedMessage.Error()
      if err != nil {
        return fmt.Errorf("Speed value message result error: %w", err)
      }
    case <-time.After(Bot_C3_Request_Timeout):
      return fmt.Errorf("Speed value message request timeout")
  }

  select {
    case comActionMessage = <-resultComActionChan:
      err = comActionMessage.Error()
      if err != nil {
        return fmt.Errorf("Speed COM_ACTION message result error: %w", err)
      }
    case <-time.After(Bot_C3_Request_Timeout):
      return fmt.Errorf("Speed COM_ACTION message request timeout")
  }

  return nil
}

func (bot *Bot) SetAXISSpeed(VEL_AXIS uint8, ACC_AXIS uint8) error {
  bot.isMovementMux.RLock()
  if bot.isMovement == true {
    bot.isMovementMux.RUnlock()
    return fmt.Errorf("Bot is movement")
  }
  bot.isMovementMux.RUnlock()

  requestSpeedVariable  := make(map[C3VariableType]*string)
  requestComActionVariable := make(map[C3VariableType]*string)

  VEL_AXISValue := fmt.Sprintf("%d", VEL_AXIS)
  requestSpeedVariable[C3Variable_COM_VALUE2] = &VEL_AXISValue

  ACC_AXISValue := fmt.Sprintf("%d", ACC_AXIS)
  requestSpeedVariable[C3Variable_COM_VALUE4] = &ACC_AXISValue

  comActionValue := string(C3Variable_COM_ACTION_VEL_AXIS)
  requestComActionVariable[C3Variable_COM_ACTION] = &comActionValue

  speedMessage, err := NewC3Message(bot.nextTagId(), requestSpeedVariable)
  if err != nil {
    return fmt.Errorf("AXISSpeed value message error: %w", err)
  }

  comActionMessage, err := NewC3Message(bot.nextTagId(), requestComActionVariable)
  if err != nil {
    return fmt.Errorf("AXISSpeed COM_ACTION message error: %w", err)
  }

  resultSpeedChan, err := bot.c3Client.Request(speedMessage)
  if err != nil {
    return fmt.Errorf("AXISSpeed value message request error: %w", err)
  }

  resultComActionChan, err := bot.c3Client.Request(comActionMessage)
  if err != nil {
    return fmt.Errorf("AXISSpeed COM_ACTION message request error: %w", err)
  }

  select {
    case speedMessage = <-resultSpeedChan:
      err = speedMessage.Error()
      if err != nil {
        return fmt.Errorf("AXISSpeed value message result error: %w", err)
      }
    case <-time.After(Bot_C3_Request_Timeout):
      return fmt.Errorf("AXISSpeed value message request timeout")
  }

  select {
    case comActionMessage = <-resultComActionChan:
      err = comActionMessage.Error()
      if err != nil {
        return fmt.Errorf("AXISSpeed COM_ACTION message result error: %w", err)
      }
    case <-time.After(Bot_C3_Request_Timeout):
      return fmt.Errorf("AXISSpeed COM_ACTION message request timeout")
  }

  return nil
}


func (bot *Bot) Move(p *Position) (bool, error) {
  bot.isMovementMux.RLock()
  if bot.isMovement == true {
    bot.isMovementMux.RUnlock()
    return false, fmt.Errorf("Bot already movement")
  }
  bot.isMovementMux.RUnlock()

  bot.positionMux.RLock()
  switch p.Type() {
    case PositionType_E6AXIS:
      if bot.c3AXIS_ACT.Equal(p, Bot_Position_Tolerance) {
        bot.positionMux.RUnlock()
        bot.isMovementMux.Lock()
        bot.isMovement = false
        bot.isMovementMux.Unlock()
        return false, nil
      }
    
    case PositionType_E6POS:
      if bot.c3POSITION.Equal(p, Bot_Position_Tolerance) {
        bot.positionMux.RUnlock()
        bot.isMovementMux.Lock()
        bot.isMovement = false
        bot.isMovementMux.Unlock()
        return false, nil
      }
      
  }
  bot.positionMux.RUnlock()

  requestPositionVariable  := make(map[C3VariableType]*string)
  requestComActionVariable := make(map[C3VariableType]*string)
  
  positionType  := p.Type()
  positionValue := p.Value()

  if positionType == PositionType_E6AXIS {
    requestPositionVariable[C3Variable_COM_E6AXIS] = &positionValue
    comActionValue := string(C3Variable_COM_ACTION_E6AXIS)
    requestComActionVariable[C3Variable_COM_ACTION] = &comActionValue
  } else if positionType == PositionType_E6POS {
    requestPositionVariable[C3Variable_COM_E6POS] = &positionValue
    comActionValue := string(C3Variable_COM_ACTION_E6POS)
    requestComActionVariable[C3Variable_COM_ACTION] = &comActionValue
  } else {
    return false, fmt.Errorf("Incorrect Move position type of %d", positionType)
  }

  comRoundmValue := string(C3Variable_COM_ROUNDM_NONE)
  requestPositionVariable[C3Variable_COM_ROUNDM] = &comRoundmValue

  positionMessage, err := NewC3Message(bot.nextTagId(), requestPositionVariable)
  if err != nil {
    return false, fmt.Errorf("Move new Position message error: %w", err)
  }

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

  bot.isMovementMux.Lock()
  bot.isMovement = true
  bot.isMovementMux.Unlock()
  log.Printf("[Bot %s INFO] Move bot to position %s\n", bot.Name, p.Value())
  
  var readyFlag uint16 = 0
  timeout := time.After(Bot_Move_Timeout)
  
  loop: for {
    select {
      case <-timeout:
        bot.isMovementMux.Lock()
        bot.isMovement = false
        bot.isMovementMux.Unlock()
        return true, fmt.Errorf("Move timeout break")
      
      default:
        switch p.Type() {
          case PositionType_E6AXIS:
            bot.positionMux.RLock()
            if bot.c3AXIS_ACT.Equal(p, Bot_Position_Tolerance) {
              readyFlag++
            }
            bot.positionMux.RUnlock()
          
          case PositionType_E6POS:
            bot.positionMux.RLock()
            if bot.c3POSITION.Equal(p, Bot_Position_Tolerance) {
              readyFlag++
            }
            bot.positionMux.RUnlock()
          
          default:
            log.Printf("[Bot %s WARNINT] Incorrect move position: %s\n", bot.Name, p.Value())
            bot.isMovementMux.Lock()
            bot.isMovement = false
            bot.isMovementMux.Unlock()
            return true, fmt.Errorf("Incorrect move position")
        }

        if readyFlag >= Bot_Position_ReadySteps {
          break loop
        }
    }
  }

  log.Printf("[Bot %s INFO] Move ready position %s\n", bot.Name, p.Value())
  bot.isMovementMux.Lock()
  bot.isMovement = false
  bot.isMovementMux.Unlock()
  return false, nil
}

func (bot *Bot) MovInternal(action uint16) (bool, error) {
  if action != 100 && action != 200 && action != 300 && action != 400 {
    return false, fmt.Errorf("Incorrect action & action != 100 && action != 200 && action != 300 && action != 400")
  }

  bot.isMovementMux.RLock()
  if bot.isMovement == true {
    bot.isMovementMux.RUnlock()
    return false, fmt.Errorf("MovInternal %d. Bot is movement", action)
  }
  bot.isMovementMux.RUnlock()

  requestComActionVariable := make(map[C3VariableType]*string)
  comActionValue := fmt.Sprintf("%d", action)
  requestComActionVariable[C3Variable_COM_ACTION] = &comActionValue

  comActionMessage, err := NewC3Message(bot.nextTagId(), requestComActionVariable)
  if err != nil {
    return false, fmt.Errorf("MovInternal %d new COM_ACTION message error: %w", action, err)
  }

  resultComActionChan, err := bot.c3Client.Request(comActionMessage)
  if err != nil {
    return false, fmt.Errorf("MovInternal %d COM_ACTION message request error: %w", action, err)
  }

  select {
    case comActionMessage = <-resultComActionChan:
      err = comActionMessage.Error()
      if err != nil {
        return false, fmt.Errorf("MovInternal %d COM_ACTION message result error: %w", action, err)
      }
    case <-time.After(Bot_C3_Request_Timeout):
      return false, fmt.Errorf("MovInternal %d COM_ACTION message request timeout", action)
  }

  bot.isMovementMux.Lock()
  bot.isMovement = true
  bot.isMovementMux.Unlock()
  log.Printf("[Bot %s INFO] MovInternal %d start\n", bot.Name, action)
  
  HOME := NewPosition(PositionType_E6AXIS)
  HOME.SetValues([14]float32{0, -90, 90, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0})

  var readyFlag uint16 = 0
  timeout := time.After(Bot_Move_Timeout)

  loop: for {
    select {
      case <-timeout:
        bot.isMovementMux.Lock()
        bot.isMovement = false
        bot.isMovementMux.Unlock()
        return true, fmt.Errorf("Move timeout break")
      
      default:
        bot.positionMux.RLock()
        if bot.c3AXIS_ACT.Equal(HOME, 0.0010) {
          readyFlag++
        }
        bot.positionMux.RUnlock()

        if readyFlag >= Bot_Position_ReadySteps {
          break loop
        }
    }
  }

  log.Printf("[Bot %s INFO] MovInternal %d ready\n", bot.Name, action)
  bot.isMovementMux.Lock()
  bot.isMovement = false
  bot.isMovementMux.Unlock()
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

  var AXIS_ACT *Position = NewPosition(PositionType_NIL)
  var POS_ACT *Position = NewPosition(PositionType_NIL)
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
  bot.c3AXIS_ACT = AXIS_ACT
  bot.c3POS_ACT = POS_ACT
  bot.c3COM_ACTION = COM_ACTION
  bot.c3COM_ROUNDM = COM_ROUNDM
  bot.c3POSITION = POS_ACT.WithOffset(bot.c3OFFSET)
  bot.positionMux.Unlock()

  return nil
}

func (bot *Bot) ResetOffsetAndPosition() error {
  bot.positionMux.Lock()
  defer bot.positionMux.Unlock()
  bot.c3OFFSET = bot.c3POS_ACT.Clone()
  bot.c3POSITION = NewPosition(PositionType_E6POS)
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
      log.Printf("[Bot %s ERROR] UpdatePosition Get position error %v\n", bot.Name, err)
      continue
    }

    if err := bot.oscResponseCurrentCoords(); err != nil {
      log.Printf("[Bot %s ERROR] Response current coords error %v\n", bot.Name, err)
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
        bot.c3PROXY_TYPE = variable.Value

      case C3Variable_PROXY_VERSION:
        bot.c3PROXY_VERSION = variable.Value

      case C3Variable_PROXY_HOSTNAME:
        bot.c3PROXY_HOSTNAME = variable.Value

      case C3Variable_PROXY_ADDRESS:
        bot.c3PROXY_ADDRESS = variable.Value

      case C3Variable_PROXY_PORT:
        bot.c3PROXY_PORT = variable.Value
    }
  }
  bot.proxyMux.Unlock()

  return nil
}

func (bot *Bot) OSCPacket(oscPacket *OSCPacket) {
  select {
    case bot.oscInput <- oscPacket:
    default:
      log.Printf("[Bot %s WARNING] OSC Input channel is full, discarding packet\n", bot.Name)
  }
}

func (bot *Bot) processOSCPackets() {
  defer bot.wg.Done()

  for packet := range bot.oscInput {
    if bot.OSCRequestAxis != nil && packet.Path == *bot.OSCRequestAxis {
      bot.processOSCAxis(packet)
      continue
    }

    if bot.OSCRequestCoords != nil && packet.Path == *bot.OSCRequestCoords {
      bot.processOSCCoords(packet)
      continue
    }

    if bot.OSCRequestPosition != nil && packet.Path == *bot.OSCRequestPosition {
      bot.processOSCPosition(packet)
      continue
    }
  }
}

func (bot *Bot) processOSCAxis(oscPacket *OSCPacket) {
  values := oscPacket.Values()
  if len(values) != 6 {
    log.Printf("[Bot %s ERROR] Incorrect OSC values length of %+v\n", bot.Name, values)
    return
  }

  position := NewPosition(PositionType_E6AXIS)
  for i, value := range values {
    switch value.(type) {
      case float32:
        position.Set(i, value.(float32))
      default:
        log.Printf("[Bot %s ERROR] OSC values[%d] is not of float32 value\n", bot.Name, i)
        return
    }
  }

  go func(position *Position) {
    if _, err := bot.Move(position); err != nil {
      log.Printf("[Bot %s ERROR] OSC Position %s move error: %v\n", bot.Name, position.Value(), err)
    }
  }(position)
}

func (bot *Bot) processOSCCoords(oscPacket *OSCPacket) {
  values := oscPacket.Values()
  if len(values) != 6 {
    log.Printf("[Bot %s ERROR] Incorrect OSC values length of %+v\n", bot.Name, values)
    return
  }

  position := NewPosition(PositionType_E6POS)
  for i, value := range values {
    switch value.(type) {
      case float32:
        position.Set(i, value.(float32))
      default:
        log.Printf("[Bot %s ERROR] OSC values[%d] is not of float32 value\n", bot.Name, i)
        return
    }
  }

  go func(position *Position) {
    if _, err := bot.Move(position); err != nil {
      log.Printf("[Bot %s ERROR] OSC Position %s move error: %v\n", bot.Name, position.Value(), err)
    }
  }(position)
}

func (bot *Bot) GetMoveGroup(id uint16) *MoveGroup {
  bot.moveGroupsMux.RLock()
  defer bot.moveGroupsMux.RLock()
  
  var moveGroup *MoveGroup = nil
  for _, moveGroup = range bot.MoveGroups {
    if moveGroup.Id == id {
      break
    }
  }

  return moveGroup
}

func (bot *Bot) MoveRound(moveGroup *MoveGroup) (bool, error) {
  if moveGroup.Id == 100 || moveGroup.Id == 200 || moveGroup.Id == 300 || moveGroup.Id == 400 {
    if isBreak, err := bot.MovInternal(moveGroup.Id); err != nil {
       return isBreak, fmt.Errorf("MoveGroup %d MoveInternal error: %w", moveGroup.Id, err)
    }

    bot.currentMoveGroupIdMux.Lock()
    bot.currentMoveGroupId =  moveGroup.Id
    bot.currentMoveGroupIdMux.Unlock()
    return false, nil
  }

  bot.currentMoveGroupIdMux.RLock()
  if err := BotETest(bot.currentMoveGroupId, moveGroup.Id); err != nil {
    bot.currentMoveGroupIdMux.RUnlock()
    return false, fmt.Errorf("MoveGroup %d Error: %w", moveGroup.Id, err)
  }
  bot.currentMoveGroupIdMux.RUnlock()
  
  for _, position := range moveGroup.Positions {
    if isBreak, err := bot.Move(position); err != nil {
      return isBreak, fmt.Errorf("MoveGroup %d Position %s move error: %w", moveGroup.Id, position.Value(), err)
    }
    time.Sleep(250 * time.Millisecond)

  }

  bot.currentMoveGroupIdMux.Lock()
  bot.currentMoveGroupId =  moveGroup.Id
  bot.currentMoveGroupIdMux.Unlock()
  return false, nil
}

func (bot *Bot) processMoveGroup() {
  defer bot.wg.Done()
  
  for moveGroup := range bot.moveInput {
    log.Printf("[Bot %s INFO] Process MoveGroup %d start\n", bot.Name, moveGroup.Id)

    if bot.isMovement == true {
      log.Printf("[Bot %s ERROR] Process MoveGroup %d skip, bot is Movement\n", bot.Name, moveGroup.Id)
      continue
    }

    if isBreak, err := bot.MoveRound(moveGroup); err != nil {
      log.Printf("[Bot %s ERROR] Process MoveGroup %d error: %v with break %v\n", bot.Name, moveGroup.Id, err, isBreak)
      continue
    }

    log.Printf("[Bot %s INFO] Process MoveGroup %d sucessful\n", bot.Name, moveGroup.Id)
  }
}

func (bot *Bot) RunMoveGroup(id uint16) error {
  moveGroup := bot.GetMoveGroup(id)
  if moveGroup == nil {
    return fmt.Errorf("MoveGroup %d in not found", id)
  }

  bot.moveInput <- moveGroup
  return nil
}

func (bot *Bot) processOSCPosition(oscPacket *OSCPacket) {
  values := oscPacket.Values()
  if len(values) != 3 {
    log.Printf("[Bot %s ERROR] Incorrect OSC Position values length of %+v\n", bot.Name, values)
    return
  }

  var id uint16 = uint16(values[0].(int32))
  var _ uint16 = uint16(values[1].(int32)) // Speed
  var index int32 = values[1].(int32)

  bot.isMovementMux.RLock()
  if bot.isMovement == true {
    bot.isMovementMux.RUnlock()
    log.Printf("[Bot %s ERROR] OSC MoveGroup error: Arready movement\n", bot.Name)
    go func() {
      if err := bot.oscResponsePosition(OSCOutputStatus_Error, index, id); err != nil {
        log.Printf("[Bot %s ERROR] OSC Response error %v\n", bot.Name, err)
      }
    }()
    return
  }
  bot.isMovementMux.RUnlock()
  
  moveGroup := bot.GetMoveGroup(id)
  if moveGroup == nil {
    log.Printf("[Bot %s ERROR] OSC MoveGroup %d in not found\n", bot.Name, id)
    go func() {
      if err := bot.oscResponsePosition(OSCOutputStatus_Error, index, id); err != nil {
        log.Printf("[Bot %s ERROR] OSC Response error %v\n", bot.Name, err)
      }
    }()
    return
  }

  go func(index int32, moveGroup *MoveGroup) {
    if isBreak, err := bot.MoveRound(moveGroup); err != nil {
      log.Printf("[Bot %s ERROR] OSC Process position error: %v\n", bot.Name, err)
      status := OSCOutputStatus_Error
      if isBreak == true {
        status = OSCOutputStatus_Break
      }
      if err := bot.oscResponsePosition(status, index, moveGroup.Id); err != nil {
        log.Printf("[Bot %s ERROR] OSC error move response error %v\n", bot.Name, err)
      }
      return
    }

    if err := bot.oscResponsePosition(OSCOutputStatus_OK, index, moveGroup.Id); err != nil {
      log.Printf("[Bot %s ERROR] OSC sucess move response error %v\n", bot.Name, err)
    }
  }(index, moveGroup)
}

func (bot *Bot) oscResponseAxis(position *Position) error {
  if bot.oscClient == nil {
    return nil
  }

  if bot.OSCResponseAxes == nil {
    return nil
  }

  return bot.oscClient.ResponseAxis(*bot.OSCResponseAxes, position)
}

func (bot *Bot) oscResponseCurrentAxis() error {
  bot.positionMux.RLock()
  defer bot.positionMux.RUnlock()
  position := bot.c3AXIS_ACT.Clone()
  return bot.oscResponseAxis(position)
}

func (bot *Bot) oscResponseCoords(position *Position) error {
  if bot.oscClient == nil {
    return nil
  }

  if bot.OSCResponseCoords == nil {
    return nil
  }

  return bot.oscClient.ResponseCoords(*bot.OSCResponseCoords, position)
}

func (bot *Bot) oscResponseCurrentCoords() error {
  bot.positionMux.RLock()
  defer bot.positionMux.RUnlock()
  position := bot.c3POSITION.Clone()
  return bot.oscResponseCoords(position)
}

func (bot *Bot) oscResponsePosition(status OSCOutputStatus, index int32, positionId uint16) error {
  if bot.oscClient == nil {
    return nil
  }

  if bot.OSCResponsePosition == nil {
    return nil
  }

  return bot.oscClient.ResponsePosition(*bot.OSCResponsePosition, status, index, positionId)
}

func (bot *Bot) GetAppData() *BotApp {
  bot.proxyMux.RLock()
  defer bot.proxyMux.RUnlock()
  bot.positionMux.RLock()
  defer bot.positionMux.RUnlock()
  bot.isMovementMux.RLock()
  defer bot.isMovementMux.RUnlock()
  bot.tagIdMux.RLock()
  defer bot.tagIdMux.RUnlock()

  botApp := &BotApp{
    Name:    bot.Name,
    Address: bot.Address,

    OSCRequestAxis:      nilStringToString(bot.OSCRequestAxis),
    OSCRequestCoords:    nilStringToString(bot.OSCRequestCoords),
    OSCRequestPosition:  nilStringToString(bot.OSCRequestPosition),

    OSCResponseAddress:  nilStringToString(bot.OSCResponseAddress),
    OSCResponseAxes:     nilStringToString(bot.OSCResponseAxes),
    OSCResponseCoords:   nilStringToString(bot.OSCResponseCoords),
    OSCResponsePosition: nilStringToString(bot.OSCResponsePosition),

    TagId:      bot.tagId,
    IsMovement: bot.isMovement,

    COM_ACTION: string(bot.c3COM_ACTION),
    COM_ROUNDM: string(bot.c3COM_ROUNDM),

    AXIS_ACT: bot.c3AXIS_ACT.Clone(),
    POS_ACT:  bot.c3POS_ACT.Clone(),
    OFFSET:   bot.c3OFFSET.Clone(),
    POSITION: bot.c3POSITION.Clone(),

    PROXY_TYPE:     bot.c3PROXY_TYPE,
    PROXY_VERSION:  bot.c3PROXY_VERSION,
    PROXY_HOSTNAME: bot.c3PROXY_HOSTNAME,
    PROXY_ADDRESS:  bot.c3PROXY_ADDRESS,
    PROXY_PORT:     bot.c3PROXY_PORT,
  }

  botApp.MoveGroups = make([]*MoveGroup, len(bot.MoveGroups))
  for i, moveGroup := range bot.MoveGroups {
    botApp.MoveGroups[i] = moveGroup.Clone()
  }

  return botApp
}

func nilStringToString(input *string) string {
  if input == nil {
    return "Ø"
  }
  return *input
}
