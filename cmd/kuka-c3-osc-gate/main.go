package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"syscall"
)

const (
	execName      = "kuka-c3-osc-gate"
	
	defaultOSCPort = 8765
  defaultConfig  = execName + ".json"
)

var (
  version       = "unknown"
  versionString = fmt.Sprintf("%s %s %s\n", execName, version, runtime.Version())
  
  logPath  = filepath.Join(os.TempDir(), execName + ".log")
)

func cli() (verboseFlag bool, oscAddr net.UDPAddr, configFile string, appPort PortValue, botInit uint) {
	var printHelp bool
  var printVersion bool
  flag.BoolVar(&printHelp, "help", false, "Print help and usage information")
  flag.BoolVar(&printVersion, "version", false, "Print version information")

  flag.BoolVar(&verboseFlag, "v", false, "Show verbose log output")

  var oscPort PortValue = defaultOSCPort
  flag.Var(&oscPort, "osc", "OSC listening port")
  oscAddr = oscPort.UDPAddr()

  appPort = PortValue_NIL
  flag.Var(&appPort, "app", "App listening port")

  flag.UintVar(&botInit, "i", 0, "Bots config init with bot count")

  configFile = filepath.Clean(*flag.String("cfg", defaultConfig, "Config file"))

  flag.Parse()

  if printVersion {
    fmt.Print(versionString)
    os.Exit(0)
  }

  if printHelp {
    fmt.Print(versionString)
    fmt.Printf("Lit3D KUKA-C3-Gate\n")
    fmt.Printf("usage: %s [options]\n\n", execName)
    fmt.Println("options:")
    flag.PrintDefaults()
    os.Exit(0)
  }

  if filepath.IsAbs(configFile) != true {
    configFile = filepath.Join("./", configFile)
  }

  return
}

func main() {
	verboseFlag, oscAddr, configFile, appPort, botInit := cli()

  if botInit > 0 {
    if err := botsConfigInit(configFile, int(botInit)); err != nil {
      fmt.Fprintf(os.Stderr, "[FATAL] Bots config init error: %v\n", err)
      os.Exit(1)
    }
    fmt.Fprintf(os.Stdout, "[INFO] Bots config init successful with %d bots\n", botInit)
    os.Exit(0)
  }

	if verboseFlag != true {
    logFile, err := os.OpenFile(logPath, os.O_RDWR | os.O_CREATE | os.O_APPEND, 0666)
    if err != nil {
    	fmt.Fprintf(os.Stderr, "[FATAL] Log file error: %v\n", err)
      os.Exit(1)
    }
    defer logFile.Close()
    log.SetOutput(logFile)
  }

  botsTeam := NewTeam(configFile)
  if err := botsTeam.Read(); err != nil {
  	log.Fatalf("[FATAL] BotTeam read error: %v\n", err)
  }

  oscServer := NewOSCServer(oscAddr)
  if err := oscServer.ListenAndServe(); err != nil {
    log.Fatalf("[FATAL] OSC Server start error: %v\n", err)
  }

  if err := botsTeam.Up(oscServer); err != nil {
    log.Fatalf("[FATAL] BotTeam start error: %v\n", err)
  }

  var app *Service = nil
  if appPort != PortValue_NIL {
    app = NewService(appPort)
    if err := app.ListenAndServe(); err != nil {
      log.Fatalf("[FATAL] App Server start error: %v\n", err)
    }
  }

  sigChan := make(chan os.Signal, 1)
  signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP, syscall.SIGQUIT)
  <-sigChan

  oscServer.UnSubscribeAll()
  oscServer.Shutdown()

  if err := botsTeam.Shutdown(); err != nil {
    log.Printf("[ERROR] BotTeam stop error: %v\n", err)
  }
  
  if app != nil {
    if err := app.Shutdown(); err != nil {
      log.Printf("[ERROR] App Server stop error: %v\n", err)
    }
  }
}

func botsConfigInit(configFile string, count int) error {
  botsTeam := NewTeam(configFile)
  for i := 0; i < count; i++ {
    bot, err := NewBot()
    if err != nil {
      return err
    }
    botsTeam.Bots = append(botsTeam.Bots, bot)
  }
  return botsTeam.Write() 
}
