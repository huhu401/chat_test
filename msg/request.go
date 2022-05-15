package msg

import (
	"encoding/binary"
	"encoding/json"
	"github.com/huhu401/chat_test/constant"
	"io"
	"log"
	"net"
	"reflect"
)

// Message 消息
type Message struct {
	Grp  uint8       // 消息组id
	Cmd  uint8       // 消息的ID
	Data interface{} // 消息的内容
}

// ReqMsgLogin 1-1
type ReqMsgLogin struct {
	RoleId int64 `json:"role_id"`
}
type RspMsgLogin struct {
	Status int32 `json:"status"`
}

// ReqMsgChat 2-1
type ReqMsgChat struct {
	Content string `json:"content"`
}
type RspMsgChat struct {
	Status int32 `json:"status"`
}

// ReqMsgJoin 2-2 进聊天室
type ReqMsgJoin struct {
	Grp int32 `json:"grp"`
}
type RspMsgJoin struct {
	Status int32 `json:"status"`
}

// RspMsgNotify 2-3 聊天消息通知
type RspMsgNotify struct {
	Msg string `json:"msg"`
}

// RspMsgHistory 2-4 聊天记录同步
type RspMsgHistory struct {
	Msg []string `json:"msg"`
}

var ReqMsgMap = map[uint16]interface{}{
	constant.MsgGrpLogin*constant.GrpBase + constant.MsgCmdLogin:  &ReqMsgLogin{},
	constant.MsgGrpChat*constant.GrpBase + constant.MsgCmdChat:    &ReqMsgChat{},
	constant.MsgGrpChat*constant.GrpBase + constant.MsgCmdJoin:    &ReqMsgJoin{},
	constant.MsgGrpChat*constant.GrpBase + constant.MsgCmdHistory: &ReqMsgJoin{},
}

var RspMsgMap = map[uint16]interface{}{
	constant.MsgGrpLogin*constant.GrpBase + constant.MsgCmdLogin:  &RspMsgLogin{},
	constant.MsgGrpChat*constant.GrpBase + constant.MsgCmdChat:    &RspMsgChat{},
	constant.MsgGrpChat*constant.GrpBase + constant.MsgCmdJoin:    &RspMsgJoin{},
	constant.MsgGrpChat*constant.GrpBase + constant.MsgCmdHistory: &RspMsgHistory{},
	constant.MsgGrpChat*constant.GrpBase + constant.MsgCmdNotify:  &RspMsgNotify{},
}

// Read 读消息
func Read(c net.Conn, svr bool) (*Message, error) {
	lenHead := make([]byte, constant.NetHeaderLen)
	_, err := io.ReadFull(c, lenHead)
	if err != nil {
		log.Println("read string error", err)
		return nil, err
	}
	dataLen := binary.BigEndian.Uint16(lenHead)
	msg := make([]byte, dataLen)
	_, err = io.ReadFull(c, msg)
	msg1 := &Message{Grp: msg[0], Cmd: msg[1]}
	var st interface{}
	if svr {
		st = ReqMsgMap[uint16(msg1.Grp)*constant.GrpBase+uint16(msg1.Cmd)]
	} else {
		st = RspMsgMap[uint16(msg1.Grp)*constant.GrpBase+uint16(msg1.Cmd)]
	}
	vT := reflect.TypeOf(st).Elem()
	newSt := reflect.New(vT).Interface()
	err = json.Unmarshal(msg[2:], newSt)
	if err != nil {
		return nil, err
	}
	msg1.Data = newSt
	return msg1, nil
}

// Write 写消息
func Write(c net.Conn, msg *Message) (int, error) {
	data, err := json.Marshal(msg.Data)
	if err != nil {
		return 0, err
	}
	data = append([]byte{0, 0, msg.Grp, msg.Cmd}, data...)
	len1 := uint16(len(data) - constant.NetHeaderLen)
	binary.BigEndian.PutUint16(data, len1)
	return c.Write(data)
}
