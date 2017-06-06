/*
des   : 读写缓存
create:
author: chh
*/
package protoserver

import (
	"encoding/binary"
	"net"
	"sync"
	. "typedefs"
	"protomsg"
	"time"
)

type NetBuf struct {
	Buf      []byte
	WOffset  int
	ROffset  int
	EndIndex int
	RWLock   *sync.RWMutex //作为输出buffer时,pushdata 和 writefd 需要保护buf
}

func (this *NetBuf) Empty() (isempty bool) {
	isempty = (this.WOffset == 0)
	return
}

const (
	BufLenMax = 512000
)

func (this *NetBuf) InitSelf(lenMax int) {

	this.Buf = make([]byte, lenMax, lenMax)
	this.EndIndex = lenMax - 1
	this.RWLock = new(sync.RWMutex)
}

func (this *NetBuf) WritableBytes() int {
	return this.EndIndex - this.WOffset + 1
}

////////////////////////////////////////////////////////////////////////////
//////////////////////  funcs for input buffer ///////////////////////////
////////////////////////////////////////////////////////////////////////////
func (this *NetBuf) ReadableBytes() int {
	return this.WOffset - this.ROffset
}

func (this *NetBuf) ReadFd(sock *net.TCPConn) (n int, err error) {
	if this.WritableBytes() < 1 {
		Logger.Println("inBuf is full, will not read data!!")
		time.Sleep(10 * time.Millisecond)
		return
	}
	n, err = sock.Read(this.Buf[this.WOffset:])
	this.WOffset += n
	if err != nil {
		Logger.Println("Error: ", err.Error())
		return
	}
	return
}

func (this *NetBuf) Parse() (msg *protomsg.PBFrame) {

	//Logger.Println("before parse buf: ROffset =", this.ROffset, "WOffset =", this.WOffset, "EndIndex =", this.EndIndex)
	msg = nil

	movetohead := false
	if this.ReadableBytes() >= protomsg.PBHeadLen {
		FrameLen := binary.BigEndian.Uint32(this.Buf[this.ROffset:this.ROffset+4])
		if FrameLen < protomsg.PBHeadLen || FrameLen > protomsg.PBFrameMaxLen {
			// 数据异常，清空
			Logger.Println("buf data error, reset buf, len = ", FrameLen)
			this.ROffset = 0
			this.WOffset = 0
			return
		}

		if this.ReadableBytes() >= int(FrameLen) {
			// 校验,字节累加和等于0
			var checkcode uint8
			for checkOffset := uint32(0); checkOffset<FrameLen;  checkOffset++{
				checkcode = checkcode + this.Buf[uint32(this.ROffset)+checkOffset]
			}

			if checkcode != 0 {
				// 此帧数据丢弃
				this.ROffset += int(FrameLen)
				Logger.Println("checkcode not 0, drop it, checkcode =", checkcode)

			} else {
				msg = new(protomsg.PBFrame)
				msg.Head.Len = FrameLen
				this.ROffset += 4
				msg.Head.Ver = uint8(this.Buf[this.ROffset])
				this.ROffset++
				msg.Head.CheckCode = uint8(this.Buf[this.ROffset])
				this.ROffset++
				msg.Head.MainID = binary.BigEndian.Uint16(this.Buf[this.ROffset:this.ROffset+2])
				this.ROffset += 2
				msg.Head.SubID = binary.BigEndian.Uint16(this.Buf[this.ROffset:this.ROffset+2])
				this.ROffset += 2
				msg.Head.SN = binary.BigEndian.Uint32(this.Buf[this.ROffset:this.ROffset+4])
				this.ROffset += 4
				msg.Head.UID = binary.BigEndian.Uint32(this.Buf[this.ROffset:this.ROffset+4])
				this.ROffset += 4


				// 拷贝数据
				BodyLen := int(FrameLen - protomsg.PBHeadLen)
				msg.Body = make([]byte, BodyLen, BodyLen)
				copy(msg.Body, this.Buf[this.ROffset:this.ROffset+BodyLen])
				this.ROffset += BodyLen
				Logger.Println("get a new msg, head info: ", msg.Head)
			}
		} else {
			// 当缓存数据不够一帧时， 要检查缓存是否已满， 满了的话把数据拷贝到头部去
			if this.ROffset + int(FrameLen) > this.EndIndex {
				movetohead = true
			}

		}
	} else {
		// 当缓存数据不够一帧时， 要检查缓存是否已满， 满了的话把数据拷贝到头部去
		if this.ROffset + protomsg.PBHeadLen > this.EndIndex {
			movetohead = true
		}

	}

	if movetohead {
		Logger.Println("inbuf is full, move data to head")
		this.WOffset = copy(this.Buf, this.Buf[this.ROffset:])
		this.ROffset = 0
	}

	if this.ROffset >= this.WOffset { //reset buf
		this.ROffset = 0
		this.WOffset = 0
	}

	//Logger.Println("after parse buf: ROffset =", this.ROffset, "WOffset =", this.WOffset, "EndIndex =", this.EndIndex)

	return
}

////////////////////////////////////////////////////////////////////////////
//////////////////////  funcs for output buffer //////////////////////////
////////////////////////////////////////////////////////////////////////////

/*
头部有多少可插入的空间
*/
func (this *NetBuf) FrontPushableBytes() int {
	return this.ROffset
}

func (this *NetBuf) PushFront(v byte) int {
	if this.FrontPushableBytes() > 0 {
		this.Buf[this.ROffset-1] = v
		this.ROffset -= 1
		return 1
	}

	return 0
}

// 输出缓冲  已弃用
func (this *NetBuf) pushData(data []byte) (n int) {
	defer func() {
		this.RWLock.Unlock()
	}()

	this.RWLock.Lock()
	if this.WOffset >= this.EndIndex {
		return 0
	}
	n = copy(this.Buf[this.WOffset:], data)
	this.WOffset += n
	return n
}

// 输出缓冲  已弃用
func (this *NetBuf) writeFd(sock *net.TCPConn) (n int, err error) {
	defer func() {
		if this.ROffset >= this.WOffset {
			this.ROffset = 0
			this.WOffset = 0
		}
		this.RWLock.RUnlock()
	}()

	this.RWLock.RLock()
	if this.ROffset >= this.WOffset {
		n = 0
		err = nil
		return
	}

	Logger.Println("write to socket from index ", this.ROffset, "to", this.WOffset-1)
	n, err = sock.Write(this.Buf[this.ROffset:this.WOffset])
	this.ROffset += n
	if err != nil {
		Logger.Println("Error: ", err.Error())
	}

	return
}
