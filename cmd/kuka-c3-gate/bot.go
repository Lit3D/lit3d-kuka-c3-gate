package main

import (
	"fmt"
	"log"
	"strconv"
	"sync"
	"time"
)

type Bot struct {
  Name    string `json:"name"`
  Address string `json:"address"`
  
  CommandPath string `json:"commandPath"`
  CoordsPath  string `json:"coordsPath"`
  
  ResponseAddress string `json:"responseAddress"`
  ResponsePath    string `json:"responsePath"`
  PositionPath    string `json:"positionPath"`

  UpdateInterval uint16 `json:"updateInterval"`

  SafeZone *SafeZone `json:"safeZone"`

  PositionsE6AXIS []E6AXIS `json:"positionsE6AXIS"`
  PositionsE6POS  []E6POS  `json:"positionsE6POS"`

  E6AXIS E6AXIS
  E6POS  E6POS

  comands chan *OSCCommandInputPacket
  coords  chan *OSCCoordsInputPacket

  c3Client  *C3Client
  oscClient *OSCClient

  isShutdown chan struct{}
  
  wg sync.WaitGroup
  UI *UI

  COM_ACTION int

  currentPosition int
  nextPosition int
}

func (bot *Bot) Up(oscServer *OSCServer) (err error) {
  bot.E6AXIS = E6AXIS{}
  bot.E6POS  = E6POS{}

  if bot.c3Client, err = NewC3Client(bot.Address); err != nil {
    return fmt.Errorf("C3Client creation error: %w", err)
  }

  if bot.oscClient, err = NewOSCClient(bot.Address); err != nil {
    return fmt.Errorf("OSCClient creation error: %w", err)
  }

  bot.comands = oscServer.CommandsSubscribe(bot.CommandPath)
  bot.coords = oscServer.CoordsSubscribe(bot.CoordsPath)
  
  bot.isShutdown = make(chan struct{})

  bot.wg.Add(1)
  go bot.processVariable()

  bot.wg.Add(1)
  go bot.processCommands()

  bot.wg.Add(1)
  go bot.processCoords()

  requestVariable := make(map[string]*string)
  requestVariable["@PROXY_TYPE"] = nil
  requestVariable["@PROXY_VERSION"] = nil
  requestVariable["@PROXY_HOSTNAME"] = nil
  requestVariable["@PROXY_ADDRESS"] = nil
  requestVariable["@PROXY_PORT"] = nil
  bot.c3Client.Request(requestVariable)

  bot.wg.Add(1)
  go bot.updateStateLoop()

  bot.updateUI()
  return nil
}

func (bot *Bot) updateUI() {
  if bot.UI == nil {
    return
  }
  bot.UI.UpdateBot(bot.Name, bot.UILines())
}

func (bot *Bot) UILines() []string {
  lines := make([]string, 5)
  lines[0] = fmt.Sprintf("%s   Address: %s   COM_ACTION: %d", bot.Name, bot.Address, bot.COM_ACTION)
  lines[1] = bot.E6AXIS.Value()
  lines[2] = bot.E6POS.Value()
  lines[4] = fmt.Sprintf("Positions (Current: %d  Next: %d):", bot.currentPosition, bot.nextPosition)
  for _, position := range bot.PositionsE6AXIS {
    lines = append(lines, fmt.Sprintf(" %d: %s", position.Id, position.Value()))
  }
  for _, position := range bot.PositionsE6POS {
    lines = append(lines, fmt.Sprintf(" %d: %s", position.Id, position.Value()))
  }
  return lines
}

func (bot *Bot) updateStateLoop() {
  defer bot.wg.Done()
  
  ticker := time.NewTicker(time.Duration(bot.UpdateInterval) * time.Microsecond)

  requestVariable := make(map[string]*string)
  requestVariable["@PROXY_TYPE"] = nil
  requestVariable["@PROXY_VERSION"] = nil
  requestVariable["@PROXY_HOSTNAME"] = nil
  requestVariable["@PROXY_ADDRESS"] = nil
  requestVariable["@PROXY_PORT"] = nil

  for {
    select {
      case <-ticker.C:
        bot.c3Client.Request(requestVariable)
      case <-bot.isShutdown:
        ticker.Stop()
        return
    }
  }
}

func (bot *Bot) processVariable() {
  defer bot.wg.Done()
  for variable := range bot.c3Client.Variables {
    if variable.ErrorCode != C3Message_Error_Success {
      log.Printf("[BOT WARNING] Variable %s error %s\n", variable.Name, C3ErrorString[variable.ErrorCode])
      continue
    }

    switch variable.Name {
      case "$AXIS_ACT":
        if err := bot.E6AXIS.Parse(variable.Value); err != nil {
          log.Printf("[BOT ERROR] Variable %s parse error: %v\n", variable.Name, err)
        }

      case "$POS_ACT":
        if err := bot.E6POS.Parse(variable.Value); err != nil {
          log.Printf("[BOT ERROR] Variable %s parse error: %v\n", variable.Name, err)
        }

      case "COM_ACTION":
        intValue, err := strconv.ParseUint(variable.Value, 10, 8)
        if err != nil {
          log.Printf("[BOT ERROR] Variable %s parse error: %v", variable.Name, err)
        }
        bot.COM_ACTION = int(intValue)

      case "@PROXY_TYPE", "@PROXY_VERSION", "@PROXY_HOSTNAME", "@PROXY_TIME", "@PROXY_ADDRESS", "@PROXY_PORT", "@PROXY_ENABLED":
        log.Printf("[BOT INFO] %s = %s\n", variable.Name, variable.Value)

      case "PING":
        continue

      default:
        log.Printf("[BOT WARNING] Unsupported variable %s with value %s\n", variable.Name, variable.Value)
    }
  }
  bot.updateUI()
}

func (bot *Bot) processCommands() {
  defer bot.wg.Done()
  for comand := range bot.comands {
    bot.nextPosition = int(comand.Position)
    log.Printf("BOT Command: %+v\n", comand)
  }
}

func (bot *Bot) processCoords() {
  defer bot.wg.Done()
  for coordsValue := range bot.coords {
    log.Printf("BOT Coords: %+v\n", coordsValue)
  }
}

func (bot *Bot) Shutdown() {
  close(bot.isShutdown)
  bot.oscClient.Shutdown()
  bot.c3Client.Shutdown()
  bot.wg.Wait()
  log.Printf("[Bot INFO] Bot shutdown successfully\n")
}

