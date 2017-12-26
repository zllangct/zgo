package pool

import (
	"errors"
	"github.com/viphxin/xingo/iface"
)

var (
	//ErrClosed 连接池已经关闭Error
	ErrClosed = errors.New("pool is closed")
)

//Pool 基本方法
type Pool interface {
	Get() (iface.IWriter, error)

	Put(iface.IWriter) error

	Close(iface.IWriter) error

	Release()

	Len() int
}
