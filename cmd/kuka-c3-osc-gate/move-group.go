package main

type MoveGroup struct {
  Id        uint16      `json:"id"`
  Positions []*Position `json:"positions"`
}

func NewMoveGroup(id uint16) *MoveGroup {
  return &MoveGroup{
    Id: id,
    Positions: make([]*Position, 0),
  }
}

func (mg *MoveGroup) Clone() *MoveGroup {
  newMG := &MoveGroup{
    Id: mg.Id,
    Positions: make([]*Position, len(mg.Positions)),
  }

  for i, position := range mg.Positions {
    newMG.Positions[i] = position.Clone()
  }

  return newMG
}