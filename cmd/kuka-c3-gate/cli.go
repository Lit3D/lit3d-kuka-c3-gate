package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
)

const (
  defaultOSCPort = 8765
  defaultBotsConfig = "bots.json"
)

type PortValue uint16

func (i *PortValue) String() string {
  return fmt.Sprint(*i)
}

func (i *PortValue) Set(s string) error {
  v, err := strconv.ParseUint(s, 10, 16)
  if err != nil {
    return err
  }
  *i = PortValue(v)
  return nil
}

func cli() (initConfig int, verboseOutput bool, debugOutput bool, oscPort PortValue, botsCofig string) {
  var printHelp bool
  var printVersion bool
  flag.BoolVar(&printHelp, "help", false, "Print help and usage information")
  flag.BoolVar(&printVersion, "version", false, "Print version information")

  flag.IntVar(&initConfig, "i", 0, "Init new config with bots count")
  flag.BoolVar(&verboseOutput, "v", false, "Show verbose log output")
  flag.BoolVar(&debugOutput, "d", false, "Show debug log output")
  
  oscPort = defaultOSCPort
  flag.Var(&oscPort, "osc", "OSC listening port")
  
  botsCofigFlag := flag.String("bots", defaultBotsConfig, "Bots config file")
  
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

  if filepath.IsAbs(*botsCofigFlag) {
    botsCofig = filepath.Clean(*botsCofigFlag)
  } else {
    botsCofig = filepath.Clean(filepath.Join("./", *botsCofigFlag))
  }

  return 
}