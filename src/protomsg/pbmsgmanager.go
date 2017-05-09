/*
des   : PB消息工厂，以及回调管理
create:
author: chh
*/
package protomsg

import (
	"sync"
	. "typedefs"
	"reflect"
	"github.com/golang/protobuf/proto"
	"encoding/binary"
)

type PBType reflect.Type
type PBHandler func(interface{}, string) bool

type PBMsgManager struct {

	MapID2Msg 	map[uint32]PBType	//ID-消息
	MapName2Msg 	map[string]PBType	//名称-消息
	MapName2ID  	map[string]uint32	//名称-ID
	MapID2Handler 	map[uint32]PBHandler	//ID-回调

}

const (
	PBHeadLen    = 18
	PBFrameMaxLen = 8192
)

// PB头部  长度18Bytes
type PBHead struct {
	Len uint32
	Ver uint8
	CheckCode uint8
	MainID uint16
	SubID uint16
	SN uint32
	UID uint32
}

type PBFrame struct {
	Head	PBHead
	Body	[]byte
}

/*
单例
*/
var pbmngr *PBMsgManager
var once sync.Once
func GetPBMsgManager() *PBMsgManager {
	once.Do(func() {
		pbmngr = &PBMsgManager{}
		pbmngr.Init()
		pbmngr.RegAllMsg()
	})
	return pbmngr
}

func MakeBigID(mainid uint16, subid uint16) (bigid uint32) {
	bigid = uint32(mainid) << 16 + uint32(subid)
	return
}

func (mngr *PBMsgManager)Init() {
	mngr.MapID2Msg = make(map[uint32]PBType)
	mngr.MapName2Msg = make(map[string]PBType)
	mngr.MapName2ID = make(map[string]uint32)
	mngr.MapID2Handler = make(map[uint32]PBHandler)
}

// 注册消息
func (mngr *PBMsgManager)RegMsg(mainId uint16, subId uint16, msgCreator interface{}) (ret bool) {

	// 生成32位ID
	msgName := reflect.TypeOf(msgCreator).Elem().String()
	id32 := MakeBigID(mainId, subId)
	pbtype := reflect.TypeOf(msgCreator).Elem()
	mngr.MapName2ID[msgName] = id32
	mngr.MapID2Msg[id32] = pbtype
	mngr.MapName2Msg[msgName] = pbtype

	return

}

// 消息注册表
func (mngr *PBMsgManager)RegAllMsg() {

	mngr.RegMsg(uint16(PB_MAIN_ID_PB_MAIN_COMM), uint16(PB_SUB_COMM_PB_SUB_COMM_HEART_BEAT), &PBHeartBeat{})


	// 最后打印一下
	//Logger.Println("registed msg:",mngr.MapName2ID,mngr.MapID2Msg,mngr.MapName2Msg)

}

// 注册处理函数
// msgName: packname.MessageName
func (mngr *PBMsgManager)RegMsgHandle(msgName string, poHandler PBHandler) (ret bool) {

	id, exist := mngr.MapName2ID[msgName]
	if !exist {
		Logger.Println("RegMsgHandle failed: cant find ID for msg ", msgName)
		ret = false
		return
	}

	mngr.MapID2Handler[id] = poHandler
	ret = true
	return
}

func (mngr *PBMsgManager)GetMsgID(msgName string) (mainid uint16, subid uint16, ret bool) {

	id, exist := mngr.MapName2ID[msgName]
	if !exist {
		Logger.Println("GetMsgID failed: cant find ID for msg ", msgName)
		ret = false
		return
	}

	subid = uint16(id & 0xFFFF)
	mainid = uint16(id >> 16)
	ret = true
	return
}

// 生成消息
func (mngr *PBMsgManager) NewMsgByID(mainId uint16, subId uint16) (newmsg interface{}, ret bool) {

	// 生成32位ID
	id32 := MakeBigID(mainId, subId)
	msgtype, exist := mngr.MapID2Msg[id32]
	if !exist {
		Logger.Println("NewMsg failed: cant find msgtype for id ", id32, mainId, subId)
		ret = false
		newmsg = nil
		return
	}

	ret = true
	newmsg = reflect.New(msgtype).Interface()
	return
}

func (mngr *PBMsgManager) NewMsgByName(name string) (newmsg interface{}, ret bool) {

	msgtype, exist := mngr.MapName2Msg[name]
	if !exist {
		Logger.Println("NewMsg failed: cant find msgtype for name ", name)
		ret = false
		newmsg = nil
		return
	}

	ret = true
	newmsg = reflect.New(msgtype).Interface()
	return
}

func (mngr *PBMsgManager) DispatchPBFrame(pbframe *PBFrame, sessioninfo string) {

	if nil == pbframe {
		return
	}

	// 先找到回调
	id32 := MakeBigID(pbframe.Head.MainID, pbframe.Head.SubID)
	handler, exist := mngr.MapID2Handler[id32]
	if !exist {
		Logger.Println("DispatchPBFrame failed: cant find handler for id ", id32)
		return
	}

	// 生成消息
	newmsg, ok := mngr.NewMsgByID(pbframe.Head.MainID, pbframe.Head.SubID)
	if !ok {
		Logger.Println("DispatchPBFrame failed: cant create msg")
		return
	}

	// 填充消息
	err := proto.Unmarshal(pbframe.Body, newmsg.(proto.Message))
	if err != nil {
		Logger.Println("unmarshaling error: ", err)
		return
	}

	// 处理消息, 放到新的协程去做
	go handler(newmsg, sessioninfo)
}

// 序列化
func (frame *PBFrame) SerializeToBuf() (obuf []byte) {
	obuf = nil

	if frame.Head.Len > PBFrameMaxLen || frame.Head.Len < PBHeadLen {
		Logger.Println("SerializeToBuf error: wrong len", frame.Head.Len)
		return
	}

	frame.Head.CheckCode = 0

	obuf = make([]byte, frame.Head.Len, frame.Head.Len)
	if obuf == nil {
		return
	}

	woffset := 0
	binary.BigEndian.PutUint32(obuf[woffset:woffset+4], frame.Head.Len)
	woffset += 4
	obuf[woffset] = frame.Head.Ver
	woffset += 1
	obuf[woffset] = frame.Head.CheckCode
	checkcodeoffset := woffset
	woffset += 1
	binary.BigEndian.PutUint16(obuf[woffset:woffset+2], frame.Head.MainID)
	woffset += 2
	binary.BigEndian.PutUint16(obuf[woffset:woffset+2], frame.Head.SubID)
	woffset += 2
	binary.BigEndian.PutUint32(obuf[woffset:woffset+4], frame.Head.SN)
	woffset += 4
	binary.BigEndian.PutUint32(obuf[woffset:woffset+4], frame.Head.UID)
	woffset += 4

	copy(obuf[woffset:], frame.Body)

	var checkcode byte
	for checkOffset := uint32(0); checkOffset<frame.Head.Len;  checkOffset++{
		checkcode = checkcode + obuf[checkOffset]
	}
	// 求和后取反
	obuf[checkcodeoffset] = ^checkcode + 1

	return
}
