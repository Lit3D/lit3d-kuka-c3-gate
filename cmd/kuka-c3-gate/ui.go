package main

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

const (
  UI_Default_LogLines = 20
  UI_Default_Tick = 1 * time.Second
  UI_Default_BotLineWidth = 96
)

type UI struct {
  botNameList  []string
  botLinesList [][]string
  botMaxLines  int
  
  logBuffer   []string
  maxLogLines int

  ticker *time.Ticker

  updateMutex sync.RWMutex 
  
  wg sync.WaitGroup
}

func NewUI() *UI {
  ui := &UI{
    botNameList:  make([]string, 0),
    botLinesList: make([][]string, 0),
    
    logBuffer:   make([]string, 0),
    maxLogLines: UI_Default_LogLines,
    
    ticker: time.NewTicker(UI_Default_Tick),
  }

  ui.wg.Add(1)
  go ui.updateLoop()

  return ui
}

func (ui *UI) Write(message []byte) (n int, err error) {
  ui.updateMutex.Lock()
  defer ui.updateMutex.Unlock()

  ui.logBuffer = append(ui.logBuffer, string(message))
  if len(ui.logBuffer) > ui.maxLogLines {
    ui.logBuffer = ui.logBuffer[len(ui.logBuffer) - ui.maxLogLines:]
  }

  return len(message), nil
}

func (ui *UI) UpdateBot(name string, lines []string) {
  ui.updateMutex.Lock()

  var idx int = -1
  for i, botName := range ui.botNameList {
    if botName == name {
      idx = i
      break
    }
  }

  if idx < 0 {
    ui.botNameList = append(ui.botNameList, name)
    ui.botLinesList = append(ui.botLinesList, lines)
  } else {
    ui.botLinesList[idx] = lines
  }

  maxLines := len(lines)
  if maxLines > ui.botMaxLines {
    ui.botMaxLines = maxLines
  }

  ui.updateMutex.Unlock()
  ui.UpdateInterface()
}

func (ui *UI) UpdateInterface() {
  ui.updateMutex.Lock()
  defer ui.updateMutex.Unlock()
  
  ui.clearScreen()
  ui.moveCursor(0, 0)
  ui.drawVersion()
  ui.drawLine()
  ui.drawBotData()
  ui.drawLine()
  ui.drawLogBuffer()
}

func (ui * UI) updateLoop() {
  defer ui.wg.Done()
  for range ui.ticker.C {
    ui.UpdateInterface()
  }
}

func (ui *UI) clearScreen() {
  fmt.Print("\033[H\033[2J")
}

func (ui *UI) moveCursor(x, y int) {
  fmt.Printf("\033[%d;%dH", y + 1, x + 1)
}

func (ui *UI) drawVersion() {
  fmt.Print(versionString)
}

func (ui *UI) drawLine() {
  fmt.Println(strings.Repeat("-", UI_Default_BotLineWidth*2))
}

func (ui *UI) drawBotData() {
  for i := 0; i < ui.botMaxLines; i++ {
    var fullLine string
    for _, botLines := range ui.botLinesList {
      botLine := botLines[i]
      fullLine = fullLine + botLines[i] + strings.Repeat(" ", UI_Default_BotLineWidth - len(botLine))
    }
    fmt.Println(fullLine)
  }
}

func (ui *UI) drawLogBuffer() {
  for _, log := range ui.logBuffer {
    fmt.Print(log)
  }
}

func (ui *UI) Shutdown() {
  ui.ticker.Stop()
  ui.UpdateInterface()
}