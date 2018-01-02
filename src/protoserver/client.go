/*
des   : tcp-server
create:
author: chh
*/
package protoserver

import (
	. "prototcp/typedefs"
	"fmt"
	"net"
	"strconv"
)
const(
	Notconnect = iota   // notconnect == 0
	Connecting         // connecting == 1
	Connected        // connected == 2
)

type ProtoClient struct {
	IP       string
	Port     int
	Session  *ProtoSession
	State	 int
}

func (this *ProtoClient) Connect(ip string, port int) (e error) {
	this.State = Connecting
	this.IP = ip
	this.Port = port

	service := ip+":"+strconv.Itoa(port)
	tcpAddr, err := net.ResolveTCPAddr("tcp4", service)
	if err != nil {
		this.State = Notconnect
		e = err
		Logger.Println("ResolveTCPAddr error:", e)
		fmt.Println("ResolveTCPAddr error:", e)
		return
	}

	conn, err2 := net.DialTCP("tcp", nil, tcpAddr)
	if err2 != nil {
		this.State = Notconnect
		e = err2
		Logger.Println("DialTCP error:", e)
		fmt.Println("DialTCP error:", e)
		return
	}

	this.State = Connected
	this.Session = new(ProtoSession)
	this.Session.Init(conn)

	go this.HandleConn(this.Session)
	return
}

func (this *ProtoClient) HandleConn(s *ProtoSession) {
	defer func() {
		Logger.Println("close conn:", s.Sock.RemoteAddr().String())
		s.Sock.Close()
		s.Quit <- true
		this.State = Notconnect
	}()

	s.Run()
}

func init() {
	fmt.Println("initing ProtoClient...")
}
