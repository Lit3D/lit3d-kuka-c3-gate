package main

import (
	"fmt"
	"net"
	"strconv"
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

func (i *PortValue) UDPAddr() net.UDPAddr{
	return net.UDPAddr{
  	IP:   net.ParseIP("0.0.0.0"),
  	Port: int(*i),
  } 
}

func (i *PortValue) TCPAddr() net.TCPAddr{
	return net.TCPAddr{
  	IP:   net.ParseIP("0.0.0.0"),
  	Port: int(*i),
  } 
}