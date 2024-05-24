package main

import (
	"encoding/json"
	"fmt"
	"math"
)

type SafeZone struct {
  Vertices [8]struct{ X, Y, Z float32 }
  XMin, XMax, YMin, YMax, ZMin, ZMax float32
}

func NewSafeZone(vertices [8]struct{ X, Y, Z float32 }) SafeZone {
  safeZone := SafeZone{Vertices: vertices}
  safeZone.updateMinMax()
  return safeZone
}

func (safeZone *SafeZone) updateMinMax() {
  safeZone.XMin, safeZone.XMax = math.MaxFloat32, -math.MaxFloat32
  safeZone.YMin, safeZone.YMax = math.MaxFloat32, -math.MaxFloat32
  safeZone.ZMin, safeZone.ZMax = math.MaxFloat32, -math.MaxFloat32

  for _, vertex := range safeZone.Vertices {
    if vertex.X < safeZone.XMin {
      safeZone.XMin = vertex.X
    }
    if vertex.X > safeZone.XMax {
      safeZone.XMax = vertex.X
    }
    if vertex.Y < safeZone.YMin {
      safeZone.YMin = vertex.Y
    }
    if vertex.Y > safeZone.YMax {
      safeZone.YMax = vertex.Y
    }
    if vertex.Z < safeZone.ZMin {
      safeZone.ZMin = vertex.Z
    }
    if vertex.Z > safeZone.ZMax {
      safeZone.ZMax = vertex.Z
    }
  }
}

func (safeZone SafeZone) MarshalJSON() ([]byte, error) {
  flat := make([]float32, 0, 24)
  for _, vertex := range safeZone.Vertices {
    flat = append(flat, vertex.X, vertex.Y, vertex.Z)
  }
  return json.Marshal(flat)
}

func (safeZone *SafeZone) UnmarshalJSON(data []byte) error {
  var flat []float32
  if err := json.Unmarshal(data, &flat); err != nil {
    return err
  }

  if len(flat) != 24 {
    return fmt.Errorf("Invalid data length for SafeZone")
  }

  for i := 0; i < 8; i++ {
    safeZone.Vertices[i] = struct{ X, Y, Z float32 }{
      X: flat[i*3],
      Y: flat[i*3+1],
      Z: flat[i*3+2],
    }
  }

  safeZone.updateMinMax()
  return nil
}

func (safeZone *SafeZone) IsPointInside(X, Y, Z float32) bool {
  return X >= safeZone.XMin && X <= safeZone.XMax &&
         Y >= safeZone.YMin && Y <= safeZone.YMax &&
         Z >= safeZone.ZMin && Z <= safeZone.ZMax
}
