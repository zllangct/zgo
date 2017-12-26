package iface

import (
	"net"
)

type Iconnection interface {
	Start()
	Stop()
	GetConnection() *net.TCPConn
	GetSessionId() uint32
	Send([]byte) error
	SendBuff([]byte) error
	RemoteAddr() net.Addr
	LostConnection()
	GetProperty(string) (interface{}, error)
	SetProperty(string, interface{})
	RemoveProperty(string)
}
