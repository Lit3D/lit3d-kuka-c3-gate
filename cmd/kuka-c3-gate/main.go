package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"syscall"
)

var (
  execName      = "lit3d-engine"
  version       = "unknown"
  versionString = fmt.Sprintf("%s %s %s\n", execName, version, runtime.Version())
  
  tempDir  = os.TempDir()
  logPath  = filepath.Join(tempDir, execName + ".log")

  DEBUG = false
)

func main() {
  initConfig, verboseOutput, debugOutput, oscPort, botsCofig := cli()
  DEBUG = debugOutput

  if initConfig > 0 {
    if err := initBotsConfig(botsCofig, initConfig); err != nil {
      os.Stderr.WriteString(fmt.Sprintf("[FATAL] Init bots error: %v", err))
      os.Exit(1)
    }

    fmt.Printf("[INFO] Init config successfully in file %s\n", botsCofig)
    os.Exit(0)
  }

  if verboseOutput != true {
    logFile, err := os.OpenFile(logPath, os.O_RDWR | os.O_CREATE | os.O_APPEND, 0666)
    if err != nil {
      os.Stderr.WriteString(fmt.Sprintf("[FATAL] Log file access error: %v", err))
      os.Exit(1)
    }
    defer logFile.Close()
    log.SetOutput(logFile)
  }
  
  log.Printf(versionString)

  bots, err := parseBotsConfig(botsCofig)
  if err != nil {
    log.Fatalf("[FATAL] %v\n", err)
  }

  oscServer := NewOSCServer(oscPort)
  oscServer.ListenAndServe()

  for _, bot := range bots {
    if err := bot.Up(oscServer); err != nil {
      log.Fatalf("[FATAL] Bot %s Up failed with error %v\n", bot.Name, err)
    }
  }

  sigChan := make(chan os.Signal, 1)
  signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP, syscall.SIGQUIT)
  <-sigChan

  oscServer.Shutdown()
  for _, bot := range bots {
    bot.Shutdown()
  }
}

func parseBotsConfig(filePath string) ([]*Bot, error) {
  var bots []*Bot

  file, err := os.OpenFile(filePath, os.O_RDONLY, 0)
  if err != nil {
    if os.IsNotExist(err) {
      return bots, nil
    }
    return nil, err
  }
  defer file.Close()

  jsonDecoder := json.NewDecoder(file)
  if err = jsonDecoder.Decode(&bots); err != nil {
    return nil, fmt.Errorf("Bots config [%s] read JSON error: %w", filePath, err)
  }

  return bots, nil
}

func initBotsConfig(filePath string, count int) error {
  var bots []Bot = make([]Bot, count)
  for i := 0; i < count; i++ {
    id := i + 1
    bots[i] = Bot{
      Name:    fmt.Sprintf("Bot %d", id),
      Address: fmt.Sprintf("192.168.0.%d", id),

      CommandPath: fmt.Sprintf("/pos_%d", id),
      CoordsPath:  fmt.Sprintf("/cord_%d", id),

      ResponseAddress: fmt.Sprintf("localhost:811%d", id),
      ResponsePath:    fmt.Sprintf("/res_%d", id),
      PositionPath:    fmt.Sprintf("/rot_%d", id),

      PositionsE6AXIS: make([]E6AXIS, 0),
      PositionsE6POS:  make([]E6POS, 0),
    }
  }

  if _, err := os.Stat(filePath); err == nil {
    dir := filepath.Dir(filePath)
    fileName := filepath.Base(filePath)
    bakFilePath := filepath.Join(dir, fileName + ".bak")
    if err := os.Rename(filePath, bakFilePath); err != nil {
      return fmt.Errorf("Backup config file error: %w", err)
    }
  }

  jsonData, err := json.MarshalIndent(bots, "", "  ")
  if err != nil {
    return fmt.Errorf("Json serialization error: %w", err) 
  }

  if err := ioutil.WriteFile(filePath, jsonData, 0644); err != nil {
    return fmt.Errorf("File write error: %w", err) 
  }

  return nil
}
