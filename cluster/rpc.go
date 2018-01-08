package cluster

import (
	"github.com/zllangct/zgo/iface"
	"github.com/zllangct/zgo/logger"
	"github.com/zllangct/zgo/utils"
	"time"
	"github.com/zllangct/zgo/pool"
	"github.com/zllangct/zgo/fnet"
	"net"
	"fmt"
)
/*
	conn type 类型

	1:主要连接
	2:主动工作连接
	3:被动工作连接

	TargetType 目标类型

	1:不需要连接池
	2:需要连接池
*/
type RpcSignal int32

const (
	REQUEST_NORESULT RpcSignal = iota
	REQUEST_FORRESULT
	RESPONSE
)

type XingoRpc struct {
	conn           iface.IWriter
	workPool       pool.Pool
	workConn       chan iface.IWriter
	asyncResultMgr *AsyncResultMgr
}

func NewXingoRpc(conn iface.IWriter) *XingoRpc {
	return &XingoRpc{
		conn:           conn,
		workPool:		nil,
		workConn:		make(chan iface.IWriter,utils.GlobalObject.ConnSize),
		asyncResultMgr: AResultGlobalObj,
	}
}

func (this *XingoRpc)InitPool()  {
	remote,err5:= this.conn.GetProperty("remote")
	if err5!=nil {
		logger.Error("InitPool get remote field"+err5.Error())
	}
	logger.Info(fmt.Sprintf("Init conn pool,pool count:%d remote:%s",utils.GlobalObject.ConnSize,remote.(string)))
	targetType,err:=this.conn.GetProperty("TargetType")
	var workpool pool.Pool = nil
	if err == nil && utils.GlobalObject.ConnSize >0{
		if targetType==1 {

		}else if targetType ==2{
			ip, err1:=this.conn.GetProperty("addr")
			if err1 != nil{
				panic(err1)
			}
			//创建
			factory:=func()(iface.IWriter,error){
				addr,err2:= net.ResolveTCPAddr("tcp4",ip.(string))
				if err2 != nil{
					return nil,err2
				}
				item:=fnet.NewTcpClient(addr.IP.String(),addr.Port,utils.GlobalObject.RpcCProtoc)
				item.Start()
				rpcdata := &RpcData{
					MsgType: REQUEST_NORESULT,
					Target:  "AddChildConnPool",
					Args:    []interface{}{utils.GlobalObject.Name},
				}
				rpcpackege, err3 := utils.GlobalObject.RpcCProtoc.GetDataPack().Pack(0, rpcdata)

				if err3 == nil {
					item.Send(rpcpackege)
					return nil,err3
				} else {
					logger.Error(err3)
					return nil,err3
				}
				return item,nil
			}
			//关闭
			close:= func(v iface.IWriter)error {
				return nil
			}
			poolConfig:=&pool.PoolConfig{
				InitialCap:  int(utils.GlobalObject.ConnSize),
				MaxCap:      int(utils.GlobalObject.ConnSize),
				Factory:factory,
				Close:close,
			}
			p,err4:=pool.NewChannelPool(poolConfig)
			if err4 !=nil {
				workpool=p
			}
		}
	}
	this.workPool=workpool
}

//获取一个链接
func (this *XingoRpc) GetOneConn() iface.IWriter {
	if utils.GlobalObject.MultiConnMode {
		if len(this.workConn)>0{
			conn:=<-this.workConn
			conn.SetProperty("type",3)
			return conn
		}
		if this.workPool != nil {
			conn,err:=this.workPool.Get()
			if err == nil{
				conn.SetProperty("type",2)
				return conn
			}
		}
	}
	this.conn.SetProperty("type",1)
	return this.conn
}
/*
归还链接到链接池
*/
func (this *XingoRpc)ConnBack(conn iface.IWriter)error{
	t,err:=conn.GetProperty("type")
	if err != nil{
		return err
	}
	switch t {
	case 1:
		//主要连接,不用做处理
	case 2:
		//主动工作连接
		return this.workPool.Put(conn)
	case 3:
		//被动链接
		this.workConn<-conn
	}
	return  nil
}

func (this *XingoRpc) CallRpcNotForResultArray(target string, args []interface{}) error {
	rpcdata := &RpcData{
		MsgType: REQUEST_NORESULT,
		Target:  target,
		Args:    args,
	}
	rpcpackege, err := utils.GlobalObject.RpcCProtoc.GetDataPack().Pack(0, rpcdata)

	if err == nil {
		conn:=this.GetOneConn()
		conn.Send(rpcpackege)
		this.ConnBack(conn)
		return nil
	} else {
		logger.Error(err)
		return err
	}
}
func (this *XingoRpc) CallRpcNotForResult(target string, args ...interface{}) error {
	rpcdata := &RpcData{
		MsgType: REQUEST_NORESULT,
		Target:  target,
		Args:    args,
	}
	rpcpackege, err := utils.GlobalObject.RpcCProtoc.GetDataPack().Pack(0, rpcdata)

	if err == nil {
		conn:=this.GetOneConn()
		conn.Send(rpcpackege)
		this.ConnBack(conn)
		return nil
	} else {
		logger.Error(err)
		return err
	}
}

func (this *XingoRpc) CallRpcForResult(target string, args ...interface{}) (*RpcData, error) {
	asyncR := this.asyncResultMgr.Add()
	rpcdata := &RpcData{
		MsgType: REQUEST_FORRESULT,
		Key:     asyncR.GetKey(),
		Target:  target,
		Args:    args,
	}
	rpcpackege, err := utils.GlobalObject.RpcCProtoc.GetDataPack().Pack(0, rpcdata)
	if err == nil {
		conn:=this.GetOneConn()
		conn.Send(rpcpackege)
		this.ConnBack(conn)
		resp, err := asyncR.GetResult(3 * time.Second)
		if err == nil {
			return resp, nil
		} else {
			//超时了 或者其他原因结果没等到
			this.asyncResultMgr.Remove(asyncR.GetKey())
			return nil, err
		}
	} else {
		logger.Error(err)
		return nil, err
	}
}
func (this *XingoRpc) CallRpcForResultArray(target string, args []interface{}) (*RpcData, error) {
	asyncR := this.asyncResultMgr.Add()
	rpcdata := &RpcData{
		MsgType: REQUEST_FORRESULT,
		Key:     asyncR.GetKey(),
		Target:  target,
		Args:    args,
	}
	rpcpackege, err := utils.GlobalObject.RpcCProtoc.GetDataPack().Pack(0, rpcdata)
	if err == nil {
		conn:=this.GetOneConn()
		conn.Send(rpcpackege)
		this.ConnBack(conn)
		resp, err := asyncR.GetResult(3 * time.Second)
		if err == nil {
			return resp, nil
		} else {
			//超时了 或者其他原因结果没等到
			this.asyncResultMgr.Remove(asyncR.GetKey())
			return nil, err
		}
	} else {
		logger.Error(err)
		return nil, err
	}
}

func (this *XingoRpc)Close()  {
	this.workConn=make(chan iface.IWriter,utils.GlobalObject.ConnSize)
	this.workPool.Release()
	this.workPool=nil
}