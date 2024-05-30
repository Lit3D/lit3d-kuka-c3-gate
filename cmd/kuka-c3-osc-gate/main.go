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
  defaultConfig = execName + ".json"
)

var (
  version       = "unknown"
  versionString = fmt.Sprintf("%s %s %s\n", execName, version, runtime.Version())
  
  logPath  = filepath.Join(os.TempDir(), execName + ".log")
)

func cli() (verboseFlag bool, oscAddr net.UDPAddr, configFile string) {
	var printHelp bool
  var printVersion bool
  flag.BoolVar(&printHelp, "help", false, "Print help and usage information")
  flag.BoolVar(&printVersion, "version", false, "Print version information")

  flag.BoolVar(&verboseFlag, "v", false, "Show verbose log output")

  var oscPort PortValue = defaultOSCPort
  flag.Var(&oscPort, "osc", "OSC listening port")
  oscAddr = oscPort.UDPAddr()

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
	verboseFlag, oscAddr, configFile := cli()

	if verboseFlag != true {
    logFile, err := os.OpenFile(logPath, os.O_RDWR | os.O_CREATE | os.O_APPEND, 0666)
    if err != nil {
    	fmt.Fprintf(os.Stderr, `[FATAL] Log file error: %v`, err)
      os.Exit(1)
    }
    defer logFile.Close()
    log.SetOutput(logFile)
  }

  config := NewConfig(configFile)
  if err := config.Read(); err != nil {
  	log.Fatalf(`[FATAL] %v`, err)
  }

  oscServer := NewOSCServer(oscAddr)
  if err := oscServer.ListenAndServe(); err != nil {
    log.Fatalf("[FATAL] OSC Server start error: %v\n", err)
  }

  for _, bot := range config.Bots {
  	oscServer.Subscribe(bot)
    if err := bot.Up(); err != nil {
      log.Fatalf("[FATAL] Bot %s Up failed with error %v\n", bot.Name, err)
    }
  }

  sigChan := make(chan os.Signal, 1)
  signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP, syscall.SIGQUIT)
  <-sigChan

  oscServer.UnSubscribeAll()
  oscServer.Shutdown()
  for _, bot := range config.Bots {
    if err := bot.Down(); err != nil {
    	log.Printf("[ERROR] Bot %s Down failed with error %v\n", bot.Name, err)
    }
  }
}