package dataStruct

import "mda-traceroute-go/util"

type Message struct {
	MsgType  string `json:"msg-type"`
	Msg      string `json:"data"`
	Datetime string `json:"datetime"`
}

func NewMessage(msgType string, msg string) *Message {
	return &Message{
		MsgType:  msgType,
		Msg:      msg,
		Datetime: util.FormatNow(),
	}
}
