package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"sync"
)

const (
  Team_PacketsBuffer = 512
)

type Team struct {
  filePath string
  fileMux  sync.Mutex

  OSCRequestPosition *string `json:"oscRequestPositionPath"`

  OSCResponseAddress  *string `json:"oscResponseAddress"`
  OSCResponsePosition *string `json:"oscResponsePositionPath"`

  Bots []*Bot `json:"bots"`

  oscInput  chan *OSCPacket
  oscClient *OSCClient

  isShutdown bool
  wg sync.WaitGroup
}

func NewTeam(filePath string) *Team {
  return &Team{
    filePath: filePath,
    Bots:     make([]*Bot, 0),
  }
}

func (team *Team) Up(oscServer *OSCServer) (err error) {
  team.oscInput = make(chan *OSCPacket, Team_PacketsBuffer)
  team.isShutdown = false

  if team.OSCResponseAddress != nil {
    if team.oscClient, err = NewOSCClient(*team.OSCResponseAddress); err != nil {
      return fmt.Errorf("OSCClient creation error: %w", err)
    }
  } else {
    team.oscClient = nil
  }

  for _, bot := range team.Bots {
    if bot.OSCResponseAddress == nil && team.OSCResponseAddress != nil {
      bot.OSCResponseAddress = team.OSCResponseAddress
    }
    
    if err := bot.Up(); err != nil {
      return fmt.Errorf("Bot %s Up failed with error %v\n", bot.Name, err)
    }
    
    oscServer.Subscribe(bot)
  }

  team.wg.Add(1)
  go team.processOSCPackets()

  oscServer.Subscribe(team)

  return nil
}

func (team *Team) Shutdown() error {
  team.isShutdown = true
  close(team.oscInput)
  team.oscClient.Shutdown()

  for _, bot := range team.Bots {
    if err := bot.Shutdown(); err != nil {
      return fmt.Errorf("Bot %s Down failed with error %v\n", bot.Name, err)
    }
  }

  team.wg.Wait()
  log.Printf("[BotTeam INFO] Shutdown successfully\n")
  return nil
}

func (team *Team) Read() error {
  team.fileMux.Lock()
  defer team.fileMux.Unlock()

  file, err := os.OpenFile(team.filePath, os.O_RDONLY, 0)
  if err != nil {
    if os.IsNotExist(err) {
      return nil
    }
    return fmt.Errorf("Open file error: %w", err)
  }
  defer file.Close()

  jsonDecoder := json.NewDecoder(file)
  if err = jsonDecoder.Decode(team); err != nil {
    return fmt.Errorf("Team [%s] decode JSON error: %w", team.filePath, err)
  }

  return nil
}

func (team *Team) Write() error {
  team.fileMux.Lock()
  defer team.fileMux.Unlock()

  jsonData, err := json.MarshalIndent(team, "", "  ")
  if err != nil {
    return fmt.Errorf("JSON serialization error: %w", err) 
  }

  if err := ioutil.WriteFile(team.filePath, jsonData, 0644); err != nil {
    return fmt.Errorf("Write file error: %w", err) 
  }

  return nil
}

func (team *Team) OSCPacket(oscPacket *OSCPacket) {
  select {
    case team.oscInput <- oscPacket:
    default:
      log.Printf("[BotTeam WARNING] OSC Input channel is full, discarding packet\n")
  }
}

func (team *Team) processOSCPackets() {
  defer team.wg.Done()

  for packet := range team.oscInput {
    if team.OSCRequestPosition != nil && packet.Path == *team.OSCRequestPosition {
      team.processOSCPosition(packet)
      continue
    }
  }
}

func (team *Team) processOSCPosition(oscPacket *OSCPacket) {
  values := oscPacket.Values()
  if len(values) != 2 {
    log.Printf("[Bot ERROR] Incorrect OSC Position values length of %+v\n", values)
    return
  }

  var positionId uint16 = uint16(values[0].(int32))
  var index int32 = values[1].(int32)

  go func(index int32, positionId uint16) {

    var wg sync.WaitGroup
    errorChan := make(chan error, len(team.Bots))
    breakChan := make(chan bool, len(team.Bots))
    
    for _, bot := range team.Bots {
      wg.Add(1)
      go func(bot *Bot, positionId uint16) {
        defer wg.Done()

        position, err := bot.GetPosition(positionId)
        if err != nil {
          errorChan <- err
        }

        if isBreak, err := bot.Move(position); err != nil {
          log.Printf("[BotTeam ERROR] Bot %s OSC Position %d:%s move error: %v\n", bot.Name, position.Id(), position.Value(), err)
          errorChan <- err
          breakChan <- isBreak
        }
      }(bot, positionId)
    }

    wg.Wait()
    close(errorChan)
    close(breakChan)

    for err := range errorChan {
      if err != nil {
        status := OSCOutputStatus_Error
        loop: for isBreak := range breakChan {
          if isBreak == true {
             status = OSCOutputStatus_Break
             break loop
          }
        }
        if err := team.oscResponsePosition(status, index, positionId); err != nil {
          log.Printf("[BotTeam ERROR] OSC error move response error %v\n", err)
        }
      }
    }

    if err := team.oscResponsePosition(OSCOutputStatus_OK, index, positionId); err != nil {
      log.Printf("[Bot ERROR] OSC sucess move response error %v\n", err)
    }
  }(index, positionId)
}

func (team *Team) oscResponsePosition(status OSCOutputStatus, index int32, positionId uint16) error {
  if team.oscClient == nil {
    return nil
  }

  if team.OSCResponsePosition == nil {
    return nil
  }

  return team.oscClient.ResponsePosition(*team.OSCResponsePosition, status, index, positionId)
}