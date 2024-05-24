package main

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

type E6AXIS struct {
  Id uint8

  A1 float32
  A2 float32
  A3 float32
  A4 float32
  A5 float32
  A6 float32
}

func (p E6AXIS) MarshalJSON() ([]byte, error) {
  return json.Marshal([7]interface{}{p.Id, p.A1, p.A2, p.A3, p.A4, p.A5, p.A6})
}

func (p *E6AXIS) UnmarshalJSON(data []byte) error {
  var arr [7]interface{}
  if err := json.Unmarshal(data, &arr); err != nil {
    return err
  }

  p.Id = uint8(arr[0].(float64))

  p.A1 = float32(arr[1].(float64))
  p.A2 = float32(arr[2].(float64))
  p.A3 = float32(arr[3].(float64))
  p.A4 = float32(arr[4].(float64))
  p.A5 = float32(arr[5].(float64))
  p.A6 = float32(arr[6].(float64))

  return nil
}

func (p *E6AXIS) Parse(value string) error {
  value = strings.Trim(value, "{}")
  value = value[8:] // Remove "E6AXIS: "

  for _, part := range strings.Split(value, ", ") {
    if len(part) < 4 {
      return fmt.Errorf("E6AXIS incorrect length of: %s", part)
    }

    key := part[:2]
    value := part[3:]

    floatValue, err := strconv.ParseFloat(value, 32)
    if err != nil {
      return fmt.Errorf("E6AXIS convert error: %w", err)
    }

    switch key {
      case "A1":
        p.A1 = float32(floatValue)
      case "A2":
        p.A2 = float32(floatValue)
      case "A3":
        p.A3 = float32(floatValue)
      case "A4":
        p.A4 = float32(floatValue)
      case "A5":
        p.A5 = float32(floatValue)
      case "A6":
        p.A6 = float32(floatValue)
    }
  }

  return nil
}

func (p *E6AXIS) Value() string {
  return fmt.Sprintf("{E6AXIS: A1 %.5f, A2 %.5f, A3 %.5f, A4 %.5f, A5 %.5f, A6 %.5f}", p.A1, p.A2, p.A3, p.A4, p.A5, p.A6)
}

type E6POS struct {
  Id uint8

  X float32
  Y float32
  Z float32
  A float32
  B float32
  C float32
}

func (p E6POS) MarshalJSON() ([]byte, error) {
  return json.Marshal([7]interface{}{p.Id, p.X, p.Y, p.Z, p.A, p.B, p.C})
}

func (p *E6POS) UnmarshalJSON(data []byte) error {
  var arr [7]interface{}
  if err := json.Unmarshal(data, &arr); err != nil {
    return err
  }

  p.Id = uint8(arr[0].(float64))

  p.X = float32(arr[1].(float64))
  p.Y = float32(arr[2].(float64))
  p.Z = float32(arr[3].(float64))
  p.A = float32(arr[4].(float64))
  p.B = float32(arr[5].(float64))
  p.C = float32(arr[6].(float64))

  return nil
}

func (p *E6POS) Parse(value string) error {
  value = strings.Trim(value, "{}")
  value = value[7:] // Remove "E6POS: "

  for _, part := range strings.Split(value, ", ") {
    if len(part) < 4 {
      return fmt.Errorf("E6POS incorrect length of: %s", part)
    }

    key := part[:1]
    value := part[3:]

    floatValue, err := strconv.ParseFloat(value, 32)
    if err != nil {
      return fmt.Errorf("E6POS convert error: %w", err)
    }

    switch key {
      case "X":
        p.X = float32(floatValue)
      case "Y":
        p.Y = float32(floatValue)
      case "Z":
        p.Z = float32(floatValue)
      case "A":
        p.A = float32(floatValue)
      case "B":
        p.B = float32(floatValue)
      case "C":
        p.C = float32(floatValue)
    }
  }

  return nil
}

func (p *E6POS) Value() string {
  return fmt.Sprintf("{E6POS: X %.5f, Y %.5f, Z %.5f, A %.5f, B %.5f, C %.5f}", p.X, p.Y, p.Z, p.A, p.B, p.C)
}
