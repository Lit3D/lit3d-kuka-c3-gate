package main

import (
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"regexp"
	"strconv"
	"time"
)

type PositionType uint8

const (
  PositionType_NIL    PositionType = 0
  PositionType_E6AXIS PositionType = 1
  PositionType_E6POS  PositionType = 2

  Position_Random_Min = float32(-50)
  Position_Random_Max = float32(50)
)

type Position struct {
  valueType PositionType
  values    [14]float32
}

func init() {
  rand.Seed(time.Now().UnixNano())
}

func randomPositionValue() float32 {
  return Position_Random_Min + rand.Float32()*(Position_Random_Max - Position_Random_Min)
}

func NewPosition(valueType PositionType) *Position {
  return &Position{
    valueType: valueType,
    values: [14]float32{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
  }
}

func NewRandomPosition(valueType PositionType) *Position {
  return &Position{
    valueType: valueType,
    values: [14]float32{
      randomPositionValue(), randomPositionValue(), randomPositionValue(), randomPositionValue(), randomPositionValue(), randomPositionValue(),
      0, 0, 0, 0, 0, 0, 0, 0,
    },
  }
}

func (p Position) MarshalJSON() ([]byte, error) {
  data := [15]interface{}{}
  data[0] = p.valueType
  for i, v := range p.values {
    data[i + 1] = v
  }

  return json.Marshal(
    [15]interface{}{data[0], data[1], data[2], data[3], data[4], data[5], data[6], data[7], data[8], data[9], data[10], data[11], data[12], data[13], data[14]},
  )
}

func (p *Position) UnmarshalJSON(input []byte) error {
  var data [15]interface{}
  if err := json.Unmarshal(input, &data); err != nil {
    return err
  }

  p.valueType = PositionType(data[0].(float64))

  for i := 0; i < 14; i++ {
    p.values[i] = float32(data[i + 1].(float64))
  }

  return nil
}

func (p *Position) Type() PositionType {
  return p.valueType
}

func (p *Position) Get(i int) float32 {
  if i < 0 || i > 13 {
    return 0
  }
  return p.values[i]
}

func (p *Position) Set(i int, value float32) error {
  if i < 0 || i > 13 {
    return fmt.Errorf("Incorrect position value index of %d", i)
  }
  p.values[i] = value
  return nil
}

func (p *Position) X(value *float32) float32 {
  if value != nil {
    p.values[0] = *value
  }
  return p.values[0]
}

func (p *Position) Y(value *float32) float32 {
  if value != nil {
    p.values[1] = *value
  }
  return p.values[1]
}

func (p *Position) Z(value *float32) float32 {
  if value != nil {
    p.values[2] = *value
  }
  return p.values[2]
}

func (p *Position) A(value *float32) float32 {
  if value != nil {
    p.values[3] = *value
  }
  return p.values[3]
}

func (p *Position) B(value *float32) float32 {
  if value != nil {
    p.values[4] = *value
  }
  return p.values[4]
}

func (p *Position) C(value *float32) float32 {
  if value != nil {
    p.values[5] = *value
  }
  return p.values[5]
}

func (p *Position) S(value *float32) float32 {
  if value != nil {
    p.values[6] = *value
  }
  return p.values[6]
}

func (p *Position) T(value *float32) float32 {
  if value != nil {
    p.values[7] = *value
  }
  return p.values[7]
}

func (p *Position) A1(value *float32) float32 {
  if value != nil {
    p.values[0] = *value
  }
  return p.values[0]
}

func (p *Position) A2(value *float32) float32 {
  if value != nil {
    p.values[1] = *value
  }
  return p.values[1]
}

func (p *Position) A3(value *float32) float32 {
  if value != nil {
    p.values[2] = *value
  }
  return p.values[2]
}

func (p *Position) A4(value *float32) float32 {
  if value != nil {
    p.values[3] = *value
  }
  return p.values[3]
}

func (p *Position) A5(value *float32) float32 {
  if value != nil {
    p.values[4] = *value
  }
  return p.values[4]
}

func (p *Position) A6(value *float32) float32 {
  if value != nil {
    p.values[5] = *value
  }
  return p.values[5]
}

func (p *Position) E1(value *float32) float32 {
  if value != nil {
    p.values[8] = *value
  }
  return p.values[8]
}

func (p *Position) E2(value *float32) float32 {
  if value != nil {
    p.values[9] = *value
  }
  return p.values[9]
}

func (p *Position) E3(value *float32) float32 {
  if value != nil {
    p.values[10] = *value
  }
  return p.values[10]
}

func (p *Position) E4(value *float32) float32 {
  if value != nil {
    p.values[11] = *value
  }
  return p.values[11]
}

func (p *Position) E5(value *float32) float32 {
  if value != nil {
    p.values[12] = *value
  }
  return p.values[12]
}

func (p *Position) E6(value *float32) float32 {
  if value != nil {
    p.values[13] = *value
  }
  return p.values[13]
}

func (p *Position) Axes() string {
  return fmt.Sprintf("{E6AXIS: A1 %.5f, A2 %.5f, A3 %.5f, A4 %.5f, A5 %.5f, A6 %.5f}",
    p.values[0], p.values[1], p.values[2], p.values[3], p.values[4], p.values[5])
}

func (p *Position) AxesFull() string {
  return fmt.Sprintf("{E6AXIS: A1 %.5f, A2 %.5f, A3 %.5f, A4 %.5f, A5 %.5f, A6 %.5f, E1 %.5f, E2 %.5f, E3 %.5f, E4 %.5f, E5 %.5f, E6 %.5f}",
    p.values[0], p.values[1], p.values[2], p.values[3], p.values[4], p.values[5], p.values[8], p.values[9], p.values[10], p.values[11], p.values[12], p.values[13])
}

func (p *Position) Coords() string {
  return fmt.Sprintf("{E6POS: X %.5f, Y %.5f, Z %.5f, A %.5f, B %.5f, C %.5f}",
    p.values[0], p.values[1], p.values[2], p.values[3], p.values[4], p.values[5])
}

func (p *Position) CoordsFull() string {
  return fmt.Sprintf("{E6POS: X %.5f, Y %.5f, Z %.5f, A %.5f, B %.5f, C %.5f, S %.5f, T %.5f, E1 %.5f, E2 %.5f, E3 %.5f, E4 %.5f, E5 %.5f, E6 %.5f}",
    p.values[0], p.values[1], p.values[2], p.values[3], p.values[4], p.values[5], p.values[6], p.values[7], p.values[8], p.values[9], p.values[10], p.values[11], p.values[12], p.values[13])
}

func (p *Position) Value() string {
  switch p.valueType {
    case PositionType_E6AXIS:
      return p.Axes()
    case PositionType_E6POS:
      return p.Coords()
  }
  return "<nil>"
}

func (p *Position) ValueFull() string {
  switch p.valueType {
    case PositionType_E6AXIS:
      return p.AxesFull()
    case PositionType_E6POS:
      return p.CoordsFull()
  }
  return "<nil>"
}

var Position_RE = regexp.MustCompile(
  `^\s*\{\s*` +
  `(?:` + 
    `(E6POS):\s+` +
    `X\s+([-+]?\d*\.\d+(?:[eE][-+]?\d+)?|\d+)` +
`,\s*Y\s+([-+]?\d*\.\d+(?:[eE][-+]?\d+)?|\d+)` +
`,\s*Z\s+([-+]?\d*\.\d+(?:[eE][-+]?\d+)?|\d+)` +
`,\s*A\s+([-+]?\d*\.\d+(?:[eE][-+]?\d+)?|\d+)` +
`,\s*B\s+([-+]?\d*\.\d+(?:[eE][-+]?\d+)?|\d+)` +
`,\s*C\s+([-+]?\d*\.\d+(?:[eE][-+]?\d+)?|\d+)` +
`(?:,\s*S\s+([-+]?\d*\.\d+(?:[eE][-+]?\d+)?|\d+))?` +
`(?:,\s*T\s+([-+]?\d*\.\d+(?:[eE][-+]?\d+)?|\d+))?` +
`(?:,\s*E1\s+([-+]?\d*\.\d+(?:[eE][-+]?\d+)?|\d+))?` +
`(?:,\s*E2\s+([-+]?\d*\.\d+(?:[eE][-+]?\d+)?|\d+))?` +
`(?:,\s*E3\s+([-+]?\d*\.\d+(?:[eE][-+]?\d+)?|\d+))?` +
`(?:,\s*E4\s+([-+]?\d*\.\d+(?:[eE][-+]?\d+)?|\d+))?` +
`(?:,\s*E5\s+([-+]?\d*\.\d+(?:[eE][-+]?\d+)?|\d+))?` +
`(?:,\s*E6\s+([-+]?\d*\.\d+(?:[eE][-+]?\d+)?|\d+))?` +
  `|` +
    `(E6AXIS):\s+` +
    `A1\s+([-+]?\d*\.\d+(?:[eE][-+]?\d+)?|\d+)` +
`,\s*A2\s+([-+]?\d*\.\d+(?:[eE][-+]?\d+)?|\d+)` +
`,\s*A3\s+([-+]?\d*\.\d+(?:[eE][-+]?\d+)?|\d+)` +
`,\s*A4\s+([-+]?\d*\.\d+(?:[eE][-+]?\d+)?|\d+)` +
`,\s*A5\s+([-+]?\d*\.\d+(?:[eE][-+]?\d+)?|\d+)` +
`,\s*A6\s+([-+]?\d*\.\d+(?:[eE][-+]?\d+)?|\d+)` +
`()()` +
`(?:,\s*E1\s+([-+]?\d*\.\d+(?:[eE][-+]?\d+)?|\d+))?` +
`(?:,\s*E2\s+([-+]?\d*\.\d+(?:[eE][-+]?\d+)?|\d+))?` +
`(?:,\s*E3\s+([-+]?\d*\.\d+(?:[eE][-+]?\d+)?|\d+))?` +
`(?:,\s*E4\s+([-+]?\d*\.\d+(?:[eE][-+]?\d+)?|\d+))?` +
`(?:,\s*E5\s+([-+]?\d*\.\d+(?:[eE][-+]?\d+)?|\d+))?` +
`(?:,\s*E6\s+([-+]?\d*\.\d+(?:[eE][-+]?\d+)?|\d+))?` +
  `)`,
)

func (p *Position) Parse(value string) error {
  matches := Position_RE.FindStringSubmatch(value)
  if matches == nil || len(matches) < 7 {
    return fmt.Errorf(`Input value "%s" does not match expected format`, value)
  }

  var start int = 0
  if matches[1] == "E6POS" {
    p.valueType = PositionType_E6POS
    start = 2
  } else if matches[16] == "E6AXIS" {
    p.valueType = PositionType_E6AXIS
    start = 17
  } else {
    return fmt.Errorf(`Position RE code error for value: %s`, value)
  }
  
  for i := start; i < start + 14; i++ {
    var err error
    var value float64 = 0
    if matches[i] != "" {
      value, err = strconv.ParseFloat(matches[i], 32)
      if err != nil {
        return fmt.Errorf(`Input value [%d]"%s" parse error: %w`, i, matches[i], err)
      }
    }
    p.values[i - start] = float32(value)
  }

  return nil
}

func (p *Position) Equal(x *Position, tolerance float32) bool {
  return math.Abs(float64(p.values[0] - x.Get(0))) <= float64(tolerance) &&
         math.Abs(float64(p.values[1] - x.Get(1))) <= float64(tolerance) &&
         math.Abs(float64(p.values[2] - x.Get(2))) <= float64(tolerance) &&
         math.Abs(float64(p.values[3] - x.Get(3))) <= float64(tolerance) &&
         math.Abs(float64(p.values[4] - x.Get(4))) <= float64(tolerance) &&
         math.Abs(float64(p.values[5] - x.Get(5))) <= float64(tolerance)
}

func (p *Position) EqualFull(x *Position, tolerance float32) bool {
  return math.Abs(float64(p.values[0]  - x.Get(0) )) <= float64(tolerance) &&
         math.Abs(float64(p.values[1]  - x.Get(1) )) <= float64(tolerance) &&
         math.Abs(float64(p.values[2]  - x.Get(2) )) <= float64(tolerance) &&
         math.Abs(float64(p.values[3]  - x.Get(3) )) <= float64(tolerance) &&
         math.Abs(float64(p.values[4]  - x.Get(4) )) <= float64(tolerance) &&
         math.Abs(float64(p.values[5]  - x.Get(5) )) <= float64(tolerance) &&
         math.Abs(float64(p.values[6]  - x.Get(6) )) <= float64(tolerance) &&
         math.Abs(float64(p.values[7]  - x.Get(7) )) <= float64(tolerance) &&
         math.Abs(float64(p.values[8]  - x.Get(8) )) <= float64(tolerance) &&
         math.Abs(float64(p.values[9]  - x.Get(9) )) <= float64(tolerance) &&
         math.Abs(float64(p.values[10] - x.Get(10))) <= float64(tolerance) &&
         math.Abs(float64(p.values[11] - x.Get(11))) <= float64(tolerance) &&
         math.Abs(float64(p.values[12] - x.Get(12))) <= float64(tolerance) &&
         math.Abs(float64(p.values[13] - x.Get(13))) <= float64(tolerance)
}

func (p *Position) WithOffset(offset *Position) *Position {
  position := NewPosition(p.Type())
  for i := 0; i < 14; i++ {
    position.Set(i, p.values[i] - offset.Get(i))
  }
  return position
}

func (p *Position) Clone() *Position {
  position := NewPosition(p.Type())
  for i := 0; i < 14; i++ {
    position.Set(i, p.values[i])
  }
  return position
}
