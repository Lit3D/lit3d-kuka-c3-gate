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

  Team_C3Emelate_StartPort = 7001
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

  c3EmelateList []*C3Emelate
}

func NewTeam(filePath string) *Team {
  return &Team{
    filePath: filePath,
    Bots:     make([]*Bot, 0),
  }
}

func (team *Team) EmulateC3Servers() error {
  team.c3EmelateList = make([]*C3Emelate, len(team.Bots))
  for i := range team.c3EmelateList {
    c3Emelate := NewC3Emelate(uint16(Team_C3Emelate_StartPort + i))
    if err := c3Emelate.ListenAndServe(); err != nil {
      return err
    }
    team.c3EmelateList[i] = c3Emelate
  }
  return nil
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

  for i, bot := range team.Bots {
    if bot.OSCResponseAddress == nil && team.OSCResponseAddress != nil {
      bot.OSCResponseAddress = team.OSCResponseAddress
    }

    c3Emelate := team.c3EmelateList[i]
    if c3Emelate != nil {
      bot.Address = c3Emelate.Address()
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
  if team.oscClient != nil {
    team.oscClient.Shutdown()
  }

  for _, bot := range team.Bots {
    if err := bot.Shutdown(); err != nil {
      return fmt.Errorf("Bot %s Down failed with error %v\n", bot.Name, err)
    }
  }

  for i, c3Emelate := range team.c3EmelateList {
    if c3Emelate != nil {
      if err := c3Emelate.Shutdown(); err != nil {
        return fmt.Errorf("C3Emulate %d down failed with error %v\n", i, err)
      }
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

  team.c3EmelateList = make([]*C3Emelate, len(team.Bots))

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

  var id uint16 = uint16(values[0].(int32))
  var index int32 = values[1].(int32)

  go func(index int32, id uint16) {

    var wg sync.WaitGroup
    errorChan := make(chan error, len(team.Bots))
    breakChan := make(chan bool, len(team.Bots))
    
    for _, bot := range team.Bots {
      wg.Add(1)
      go func(bot *Bot, id uint16) {
        defer wg.Done()

        moveGroup := bot.GetMoveGroup(id)
        if moveGroup == nil {
          log.Printf("[BotTeam WARNING] Bot %s OSC MoveGroup %d is not found\n", bot.Name, id)
          return
        }

        if isBreak, err := bot.MoveRound(moveGroup); err != nil {
          log.Printf("[BotTeam ERROR] Bot %s OSC Position error: %v\n", bot.Name, err)
          errorChan <- err
          breakChan <- isBreak
        }
      }(bot, id)
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
        if err := team.oscResponsePosition(status, index, id); err != nil {
          log.Printf("[BotTeam ERROR] OSC error move response error %v\n", err)
        }
      }
    }

    if err := team.oscResponsePosition(OSCOutputStatus_OK, index, id); err != nil {
      log.Printf("[Bot ERROR] OSC sucess move response error %v\n", err)
    }
  }(index, id)
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

func (team *Team) GetAppData() []*BotApp {
  teamAppData := make([]*BotApp, len(team.Bots))
  for i, bot := range team.Bots {
    teamAppData[i] = bot.GetAppData()
  }
  return teamAppData
}
