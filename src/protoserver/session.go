/*
des   : 一个连接
create:
author: ingangi
*/
package protoserver

import (
	"prototcp/protomsg"
	. "prototcp/typedefs"
	"net"
	"reflect"
	"runtime"

	"github.com/golang/protobuf/proto"
)

type pbMsgForChan struct {
	pbmsg interface{}
	uid   uint32
	sn    uint32
}

type ProtoSession struct {
	Sock *net.TCPConn
	IBuf *NetBuf
	//OBuf    *NetBuf	//输出改用chan
	OBuf    chan pbMsgForChan
	Quit 	chan bool
	Addr    string
}

func (this *ProtoSession) Init(sock *net.TCPConn) {
	if sock == nil {
		Logger.Println("Init Error: sock is nil ")
		return
	}

	this.Sock = sock
	this.Addr = sock.RemoteAddr().String()

	this.IBuf = new(NetBuf)
	this.IBuf.InitSelf(BufLenMax)

	//this.OBuf = new(NetBuf)
	//this.OBuf.InitSelf(BufLenMax)

	this.OBuf = make(chan pbMsgForChan, 100) //允许缓存100个消息
	this.Quit = make(chan bool)
}

// 读OBuf chan, 写到socket
func (this *ProtoSession) listenOutData() {
	// 等待处理返回
	defer func() {
		Logger.Println(this.Addr, "ProtoSession listenOutData Exit")
		close(this.OBuf)
	}()

FOR:
	for {
		select {
		case rsp := <-this.OBuf:
			//Logger.Println("Get out msg")
			this.writePBMsgToSocket(&rsp)
		case <-this.Quit:
			break FOR
			//runtime.Gosched() //GOSCHED太快
		}

	}
}

func (this *ProtoSession) Run() {
	defer func() {
		Logger.Println(this.Addr, "ProtoSession Run Exit")
	}()

	//this.Running = true

	go this.listenOutData()
	//go this.SendToSock()
	for {
		_, err := this.IBuf.ReadFd(this.Sock)
		if err != nil {
			Logger.Println(this.Addr, "IBuf.ReadFd Error")
			return
		}

		msg := this.IBuf.Parse()
		for msg != nil {
			//go this.HandleMsg(msg)
			protomsg.GetPBMsgManager().DispatchPBFrame(msg, this.Addr)
			msg = this.IBuf.Parse()
		}
	}
}

// 暂时没用, 测试用
//func (this *ProtoSession) HandleMsg(msg string) {
//	defer func() {
//		Logger.Println(this.Addr, "ProtoSession HandleMsg Exit")
//	}()
//
//	Logger.Println(this.Addr, "handle msg:", msg)
//
//	//fake rsp
//	rsp := *(new(ReqContext))
//	rsp.Commd = msg
//	//rsp.Result, _ = gocache.GetMemProcesser().HandleCmd(msg)
//	this.Contex <- rsp
//}

/*
发送一个PB消息
*/
func (this *ProtoSession) writePBMsgToSocket(pomsg *pbMsgForChan) (n int, e error) {

	n = 0

	msgname := reflect.TypeOf(pomsg.pbmsg).Elem().String()
	mainid, subid, ok := protomsg.GetPBMsgManager().GetMsgID(msgname)
	if !ok {
		Logger.Println(this.Addr, "SendPBMsg faild: cant get msgid for msg name:", msgname)
		return
	}

	pbframe := new(protomsg.PBFrame)
	pbframe.Body, e = proto.Marshal(pomsg.pbmsg.(proto.Message))
	if e != nil {
		Logger.Println(this.Addr, "SendPBMsg faild: marshaling error: ", e)
		return
	}

	pbframe.Head.Len = uint32(len(pbframe.Body)) + protomsg.PBHeadLen
	pbframe.Head.Ver = 1
	pbframe.Head.SubID = subid
	pbframe.Head.MainID = mainid
	pbframe.Head.SN = pomsg.sn
	pbframe.Head.UID = pomsg.uid

	data := pbframe.SerializeToBuf()
	if data == nil {
		Logger.Println(this.Addr, "SendPBMsg faild: SerializeToBuf error")
		return
	}

	datalen := len(data)

	n, e = this.Sock.Write(data)
	if e != nil {
		Logger.Println(this.Addr, "write socket Error: ", e.Error())
	}

	Logger.Println(this.Addr, n, "bytes written,", datalen, "expected")

	// 没发完 继续写
	for n < datalen {
		Logger.Println(this.Addr,
			"writePBMsgToSocket faild:",
			n,
			"bytes written,",
			datalen-n,
			"bytes left, try again")

		n2, _ := this.Sock.Write(data[n:])
		n += n2
		runtime.Gosched()
	}

	return

}

/*
只负责把数据放到缓存, 缓存满了会阻塞，
ProtoSession只提供发送PB消息的接口
*/
func (this *ProtoSession) SendPBMsg(pbmsg interface{}, uid uint32, sn uint32) {
	this.OBuf <- pbMsgForChan{pbmsg, uid, sn}
}

/*
真正把数据发送出去
*/
//func (this *ProtoSession) SendToSock() {
//	defer func() {
//		Logger.Println(this.Addr, "ProtoSession SendToSock Exit")
//	}()
//
//	for {
//		if !this.Running {
//			return
//		} else {
//			n, e := this.OBuf.WriteFd(this.Sock)
//			if e != nil {
//				Logger.Println(this.Addr, "Error: ", e.Error())
//				return
//			}
//
//			if n == 0 {
//				runtime.Gosched()
//			}
//		}
//
//	}
//}
