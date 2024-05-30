package main

import (
  "encoding/json"
  "fmt"
  "io/ioutil"
  "os"
  "sync"
)

type Config struct {
  filePath string
  mux      sync.Mutex

  Bots []*Bot
}

func NewConfig(filePath string) *Config {
  return &Config{
    filePath: filePath,
    Bots:     make([]*Bot, 0),
  }
}

func (c *Config) Read() error {
  c.mux.Lock()
  defer c.mux.Unlock()

  file, err := os.OpenFile(c.filePath, os.O_RDONLY, 0)
  if err != nil {
    if os.IsNotExist(err) {
      return nil
    }
    return fmt.Errorf("Config open file error: %w", err)
  }
  defer file.Close()

  jsonDecoder := json.NewDecoder(file)
  if err = jsonDecoder.Decode(&(c.Bots)); err != nil {
    return fmt.Errorf("Config [%s] decode JSON error: %w", c.filePath, err)
  }

  return nil
}

func (c *Config) Write() error {
  c.mux.Lock()
  defer c.mux.Unlock()

  jsonData, err := json.MarshalIndent(c.Bots, "", "  ")
  if err != nil {
    return fmt.Errorf("Config JSON serialization error: %w", err) 
  }

  if err := ioutil.WriteFile(c.filePath, jsonData, 0644); err != nil {
    return fmt.Errorf("Config write file error: %w", err) 
  }

  return nil
}
