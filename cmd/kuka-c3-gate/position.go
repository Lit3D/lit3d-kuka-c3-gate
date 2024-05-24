package main

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
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

var E6AXIS_RE = regexp.MustCompile(`A1\s+([-+]?\d*\.\d+|\d+),\s*A2\s+([-+]?\d*\.\d+|\d+),\s*A3\s+([-+]?\d*\.\d+|\d+),\s*A4\s+([-+]?\d*\.\d+|\d+),\s*A5\s+([-+]?\d*\.\d+|\d+),\s*A6\s+([-+]?\d*\.\d+|\d+E?[-+]?\d*)`)

func (p *E6AXIS) Parse(value string) error {
  matches := E6AXIS_RE.FindStringSubmatch(value)
  if matches == nil || len(matches) < 7 {
    return fmt.Errorf("Input value '%s' does not match expected format", value)
  }

  A1, err := strconv.ParseFloat(matches[1], 32)
  if err != nil {
    return fmt.Errorf("Input value A1 parse error: %w", err)
  }
  p.A1 = float32(A1)

  A2, err := strconv.ParseFloat(matches[2], 32)
  if err != nil {
    return fmt.Errorf("Input value A2 parse error: %w", err)
  }
  p.A2 = float32(A2)

  A3, err := strconv.ParseFloat(matches[3], 32)
  if err != nil {
    return fmt.Errorf("Input value A3 parse error: %w", err)
  }
  p.A3 = float32(A3)

  A4, err := strconv.ParseFloat(matches[4], 32)
  if err != nil {
    return fmt.Errorf("Input value A4 parse error: %w", err)
  }
  p.A4 = float32(A4)

  A5, err := strconv.ParseFloat(matches[5], 32)
  if err != nil {
    return fmt.Errorf("Input value A5 parse error: %w", err)
  }
  p.A5 = float32(A5)

  A6, err := strconv.ParseFloat(matches[6], 32)
  if err != nil {
    return fmt.Errorf("Input value A6 parse error: %w", err)
  }
  p.A6 = float32(A6)

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

var E6POS_RE = regexp.MustCompile(`X\s+([-+]?\d*\.\d+|\d+),\s*Y\s+([-+]?\d*\.\d+|\d+),\s*Z\s+([-+]?\d*\.\d+|\d+),\s*A\s+([-+]?\d*\.\d+|\d+),\s*B\s+([-+]?\d*\.\d+|\d+),\s*C\s+([-+]?\d*\.\d+|\d+E?[-+]?\d*)`)

func (p *E6POS) Parse(value string) error {
  matches := E6POS_RE.FindStringSubmatch(value)
  if matches == nil || len(matches) < 7 {
    return fmt.Errorf("Input value '%s' does not match expected format", value)
  }

  X, err := strconv.ParseFloat(matches[1], 32)
  if err != nil {
    return fmt.Errorf("Input value X parse error: %w", err)
  }
  p.X = float32(X)

  Y, err := strconv.ParseFloat(matches[2], 32)
  if err != nil {
    return fmt.Errorf("Input value Y parse error: %w", err)
  }
  p.Y = float32(Y)

  Z, err := strconv.ParseFloat(matches[3], 32)
  if err != nil {
    return fmt.Errorf("Input value Z parse error: %w", err)
  }
  p.Z = float32(Z)

  A, err := strconv.ParseFloat(matches[4], 32)
  if err != nil {
    return fmt.Errorf("Input value A parse error: %w", err)
  }
  p.A = float32(A)

  B, err := strconv.ParseFloat(matches[5], 32)
  if err != nil {
    return fmt.Errorf("Input value B parse error: %w", err)
  }
  p.B = float32(B)

  C, err := strconv.ParseFloat(matches[6], 32)
  if err != nil {
    return fmt.Errorf("Input value C parse error: %w", err)
  }
  p.C = float32(C)

  return nil
}

func (p *E6POS) Value() string {
  return fmt.Sprintf("{E6POS: X %.5f, Y %.5f, Z %.5f, A %.5f, B %.5f, C %.5f}", p.X, p.Y, p.Z, p.A, p.B, p.C)
}
