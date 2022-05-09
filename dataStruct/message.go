package dataStruct

type Message struct {
	MsgType  string `json:"msg-type"`
	Msg      string `json:"data"`
	Datetime string `json:"datetime"`
}
