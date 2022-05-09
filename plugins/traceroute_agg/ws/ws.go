package ws

import (
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"mda-traceroute-go/dataStruct"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	uuid "github.com/satori/go.uuid"
)

// Manager 所有 websocket 信息
type Manager struct {
	Group                   map[string]map[string]*Client
	clientCountInGroup      map[string]uint
	groupCount, clientCount uint
	Lock                    sync.Mutex
	Register, UnRegister    chan *Client
	Message                 chan *MessageData
	GroupMessage            chan *GroupMessageData
	BroadCastMessage        chan *BroadCastMessageData
}

// Client 单个 websocket 信息
type Client struct {
	Id     string
	Group  string
	Socket *websocket.Conn
	// 待发送消息
	ToBeSentMessage chan []byte
	// 待读取消息 接收的消息放入此通道
	ToBeReadMessage chan []byte
}

// MessageData 单个发送数据信息
type MessageData struct {
	Id      string
	Group   string
	Message []byte
}

// GroupMessageData 组广播数据信息
type GroupMessageData struct {
	Group   string
	Message []byte
}

// BroadCastMessageData 广播发送数据信息
type BroadCastMessageData struct {
	Message []byte
}

// Start 启动 Websocket 管理器
func (manager *Manager) Start() {
	logrus.Infof("websocket manage start")
	for {
		select {
		// 注册
		case client := <-manager.Register:
			manager.Lock.Lock()
			logrus.Infof("client [%s] connect", client.Id)
			logrus.Infof("register client [%s] to group [%s]", client.Id, client.Group)
			if manager.Group[client.Group] == nil {
				manager.Group[client.Group] = make(map[string]*Client)
				manager.groupCount += 1
				manager.clientCountInGroup[client.Group] = 0
			}
			manager.clientCountInGroup[client.Group] += 1
			manager.Group[client.Group][client.Id] = client
			manager.clientCount += 1
			manager.Lock.Unlock()

		// 注销
		case client := <-manager.UnRegister:
			manager.Lock.Lock()
			logrus.Infof("unregister client [%s] from group [%s]", client.Id, client.Group)
			if _, ok := manager.Group[client.Group]; ok {
				if _, ok := manager.Group[client.Group][client.Id]; ok {
					close(client.ToBeSentMessage)
					close(client.ToBeReadMessage)
					delete(manager.Group[client.Group], client.Id)
					manager.clientCountInGroup[client.Group] -= 1
					manager.clientCount -= 1
					if len(manager.Group[client.Group]) == 0 {
						//logrus.Infof("delete empty group [%s]", client.Group)
						delete(manager.Group, client.Group)
						manager.groupCount -= 1
					}
				}
			}
			manager.Lock.Unlock()

			// 发送广播数据到某个组的 channel 变量 Send 中
			//case data := <-manager.boardCast:
			//	if groupMap, ok := manager.wsGroup[data.GroupId]; ok {
			//		for _, conn := range groupMap {
			//			conn.Send <- data.Data
			//		}
			//	}
		}
	}
}

// SendService 处理单个 client 发送数据
func (manager *Manager) SendService() {
	for {
		select {
		case data := <-manager.Message:
			if groupMap, ok := manager.Group[data.Group]; ok {
				if conn, ok := groupMap[data.Id]; ok {
					conn.ToBeSentMessage <- data.Message
				}
			}
		}
	}
}

// SendGroupService 处理 group 广播数据
func (manager *Manager) SendGroupService() {
	for {
		select {
		// 发送广播数据到某个组的 channel 变量 Send 中
		case data := <-manager.GroupMessage:
			if groupMap, ok := manager.Group[data.Group]; ok {
				for _, conn := range groupMap {
					conn.ToBeSentMessage <- data.Message
				}
			}
		}
	}
}

// SendAllService 处理广播数据
func (manager *Manager) SendAllService() {
	for {
		select {
		case data := <-manager.BroadCastMessage:
			for _, v := range manager.Group {
				for _, conn := range v {
					conn.ToBeSentMessage <- data.Message
				}
			}
		}
	}
}

// Send 向指定的 client 发送数据
func (manager *Manager) Send(id string, group string, message []byte) {
	data := &MessageData{
		Id:      id,
		Group:   group,
		Message: message,
	}
	manager.Message <- data
}

// SendGroup 向指定的 Group 广播
func (manager *Manager) SendGroup(group string, message []byte) {
	data := &GroupMessageData{
		Group:   group,
		Message: message,
	}
	manager.GroupMessage <- data
}

// SendAll 广播
func (manager *Manager) SendAll(message []byte) {
	data := &BroadCastMessageData{
		Message: message,
	}
	manager.BroadCastMessage <- data
}

// RegisterClient 注册
func (manager *Manager) RegisterClient(client *Client) {
	manager.Register <- client
}

// UnRegisterClient 注销
func (manager *Manager) UnRegisterClient(client *Client) {
	manager.UnRegister <- client
}

// LenGroup 当前组个数
func (manager *Manager) LenGroup() uint {
	return manager.groupCount
}

// LenClient 当前连接个数
func (manager *Manager) LenClient() uint {
	return manager.clientCount
}

func (manager *Manager) LenClientInGroup(group string) (uint, bool) {
	cnt, ok := manager.clientCountInGroup[group]
	return cnt, ok
}

// Info 获取 wsManager 管理器信息
func (manager *Manager) Info() map[string]interface{} {
	managerInfo := make(map[string]interface{})
	managerInfo["groupLen"] = manager.LenGroup()
	managerInfo["clientLen"] = manager.LenClient()
	managerInfo["chanRegisterLen"] = len(manager.Register)
	managerInfo["chanUnregisterLen"] = len(manager.UnRegister)
	managerInfo["chanMessageLen"] = len(manager.Message)
	managerInfo["chanGroupMessageLen"] = len(manager.GroupMessage)
	managerInfo["chanBroadCastMessageLen"] = len(manager.BroadCastMessage)
	return managerInfo
}

func (manager *Manager) NodeInfo(nodes *[]string) {
	for k, _ := range manager.Group {
		*nodes = append(*nodes, k)
	}
}

// WebsocketManager 初始化 wsManager 管理器
var WebsocketManager = Manager{
	Group:              make(map[string]map[string]*Client),
	Register:           make(chan *Client, 128),
	UnRegister:         make(chan *Client, 128),
	GroupMessage:       make(chan *GroupMessageData, 1024),
	Message:            make(chan *MessageData, 1024),
	BroadCastMessage:   make(chan *BroadCastMessageData, 1024),
	groupCount:         0,
	clientCount:        0,
	clientCountInGroup: make(map[string]uint),
}

// WsClient gin 处理 websocket handler
func (manager *Manager) WsClient(ctx *gin.Context) {
	upGrader := websocket.Upgrader{
		// cross origin domain
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
		// 处理 Sec-WebSocket-Protocol Header
		Subprotocols: []string{ctx.GetHeader("Sec-WebSocket-Protocol")},
	}

	conn, err := upGrader.Upgrade(ctx.Writer, ctx.Request, nil)
	if err != nil {
		logrus.Infof("websocket connect error: %s", ctx.Param("group"))
		return
	}

	client := &Client{
		Id:              uuid.NewV4().String(),
		Group:           ctx.Param("group"),
		Socket:          conn,
		ToBeSentMessage: make(chan []byte, 1024),
		ToBeReadMessage: make(chan []byte, 1024),
	}

	manager.RegisterClient(client)
	go client.Read()
	go client.Write()

	// 心跳检测
	go client.heartbeat()
}

// 读信息，从 websocket 连接直接读取数据
func (c *Client) Read() {
	defer func() {
		WebsocketManager.UnRegister <- c
		logrus.Infof("client [%s] disconnect", c.Id)
		if err := c.Socket.Close(); err != nil {
			logrus.Infof("client [%s] disconnect err: %s", c.Id, err)
		}
	}()

	for {
		messageType, message, err := c.Socket.ReadMessage()
		if err != nil || messageType == websocket.CloseMessage {
			break
		}
		logrus.Infof("receive client[%s] message: %s", c.Id, string(message))
		c.ToBeReadMessage <- message
	}
}

// 写信息，从管道中读取数据写入 websocket 连接
func (c *Client) Write() {
	defer func() {
		// 写入失败，注销该 Client
		WebsocketManager.UnRegister <- c
		logrus.Infof("client [%s] disconnect", c.Id)
		if err := c.Socket.Close(); err != nil {
			logrus.Infof("client [%s] disconnect err: %s", c.Id, err)
		}
	}()

	for {
		select {
		case message, ok := <-c.ToBeSentMessage:
			if !ok {
				_ = c.Socket.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			logrus.Infof("send to client[%s] message: %s", c.Id, string(message))
			err := c.Socket.WriteMessage(websocket.BinaryMessage, message)
			if err != nil {
				logrus.Errorf("send to client[%s] err: %s", c.Id, err)
				return
			}
		}
	}
}

// TestSendGroup 测试组广播
func TestSendGroup() {
	for {
		time.Sleep(time.Second * 20)
		WebsocketManager.SendGroup("leffss", []byte("SendGroup message ----"+time.Now().Format("2006-01-02 15:04:05")))
	}
}

// TestSendAll 测试广播
func TestSendAll() {
	for {
		time.Sleep(time.Second * 25)
		WebsocketManager.SendAll([]byte("SendAll message ----" + time.Now().Format("2006-01-02 15:04:05")))
		fmt.Println(WebsocketManager.Info())
	}
}

func (c *Client) heartbeat() {
	defer func() {
		if recover() != nil {
			logrus.Errorf("client's ToBeSentMessage chan is closed, heartbeat() sign out.")
		}
	}()

	for {
		testMsg := &dataStruct.Message{
			MsgType:  "heartbeat",
			Datetime: time.Now().Format("2006-01-02 15:04:05"),
		}
		msg, err := json.Marshal(testMsg)
		if err != nil {
			logrus.Errorf("%v", err)
		}
		c.ToBeSentMessage <- msg

		// 心跳频率 60s一次
		time.Sleep(time.Second * 60)
	}
}
