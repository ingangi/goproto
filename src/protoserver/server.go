/*
des   : tcp-server
create:
author: chh
*/
package protoserver

import (
	"fmt"
	"net"
	. "typedefs"
	"protomsg"
)

type ProtoServer struct {
	IP       string
	Port     int
	Sessions map[string]*ProtoSession
	Listener *net.TCPListener
}

func (this *ProtoServer) Listen(ip string, port int) (e error) {
	this.IP = ip
	this.Port = port
	this.Sessions = make(map[string]*ProtoSession)

	tcpAddr, _ := net.ResolveTCPAddr("tcp", fmt.Sprintf("%s:%d", ip, port))

	fmt.Println("try to listen on:", tcpAddr)

	this.Listener, e = net.ListenTCP("tcp", tcpAddr)

	//this.Listener, e = net.Listen("tcp", fmt.Sprintf("%s:%d", ip, port))
	if e != nil {
		Logger.Println("listen error:", e)
		fmt.Println("listen error:", e)
	}
	return
}

func (this *ProtoServer) RegMsgHandler() {
	protomsg.GetPBMsgManager().RegMsgHandle("protomsg.PBHeartBeat", this.OnHeartBeat)
}

func (this *ProtoServer) OnHeartBeat(pbmsg interface{}, sessioninfo string) bool {
	Logger.Println("get heartbeat from", sessioninfo, ", send back")

	hbmsg := pbmsg.(*protomsg.PBHeartBeat)
	if hbmsg.GetICurStep() != int32(protomsg.PB_EN_MSG_PROCESS_STEP_EN_MSG_PROCESS_STEP_INIT) {
		return false
	}

	c, exist := this.Sessions[sessioninfo]
	if !exist {
		Logger.Println("OnHeartBeat failed: cant find session for ", sessioninfo)
		return false
	}

	hbmsg.ICurStep = int32(protomsg.PB_EN_MSG_PROCESS_STEP_EN_MSG_PROCESS_STEP_SYNC)
	c.SendPBMsg(hbmsg, 0, 0)
	return true
}

func (this *ProtoServer) Run() {
	if this.Listener != nil {
		this.RegMsgHandler()
		for {
			c, err := this.Listener.AcceptTCP()
			if err != nil {
				fmt.Println("accept error:", err)
				break
			}

			var NewSession *ProtoSession = new(ProtoSession)
			NewSession.Init(c)
			this.Sessions[c.RemoteAddr().String()] = NewSession
			Logger.Println("new connection:", c.RemoteAddr().String())
			go this.HandleConn(NewSession)

		}
	} else {
		Logger.Println("ProtoServer Listener is nil, exit")
		fmt.Println("ProtoServer Listener is nil, exit")
	}

	Logger.Println("ProtoServer exiting...")
	fmt.Println("ProtoServer exiting...")
}

func (this *ProtoServer) HandleConn(s *ProtoSession) {
	defer func() {
		Logger.Println("close conn:", s.Sock.RemoteAddr().String())
		s.Sock.Close()
		delete(this.Sessions, s.Sock.RemoteAddr().String())
		s.Quit <- true
	}()

	s.Run()
}

func init() {
	fmt.Println("initing protoserver...")
	// e := ProtoServer.Listen(ServerConfig.ServerIP, ServerConfig.ServerPort)
	// if e != nil {
	// 	Logger.Println("listen error:", e)
	// }
}

/*
just a test func
*/
func SayHello() {
	Logger.Println("Hello")
}
