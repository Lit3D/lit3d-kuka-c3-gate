package main

type BotApp struct {
	Name    string `json:"name"`
  Address string `json:"address"`

  OSCRequestAxis     string `json:"oscRequestAxis"`
  OSCRequestCoords   string `json:"oscRequestCoords"`
  OSCRequestPosition string `json:"oscRequestPosition"`

  OSCResponseAddress string `json:"oscResponseAddress"`
  
  OSCResponseAxes     string `json:"oscResponseAxes"`
  OSCResponseCoords   string `json:"oscResponseCoords"`
  OSCResponsePosition string `json:"oscResponsPosition"`

  MoveGroups []*MoveGroup `json:"moveGroups"`

  TagId 		 uint16 `json:"tagID"`
  IsMovement bool   `json:"isMovement"`

  COM_ACTION string `json:"COM_ACTION"`
 	COM_ROUNDM string `json:"COM_ROUNDM"`

  AXIS_ACT   *Position `json:"AXIS_ACT"`
  POS_ACT    *Position `json:"POS_ACT"`
 	OFFSET     *Position `json:"OFFSET"`
 	POSITION   *Position `json:"POSITION"`

 	PROXY_TYPE     string `json:"PROXY_TYPE"`
  PROXY_VERSION  string `json:"PROXY_VERSION"`
 	PROXY_HOSTNAME string `json:"PROXY_HOSTNAME"`
 	PROXY_ADDRESS  string `json:"PROXY_ADDRESS"`
 	PROXY_PORT     string `json:"PROXY_PORT"`
} 


