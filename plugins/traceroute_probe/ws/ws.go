package ws

import (
	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
	"mda-traceroute-go/plugins/traceroute_probe/utils"
	"net/url"
	"strconv"
)

func ConnWsServer(server string, port uint16, read chan<- []byte, write <-chan []byte) *websocket.Conn {
	var addr = server + ":" + strconv.Itoa(int(port))
	u := url.URL{Scheme: "ws", Host: addr, Path: "/ws/" + utils.ConfigData.WebSocketConf.Group}
	var dialer *websocket.Dialer

	conn, _, err := dialer.Dial(u.String(), nil)
	if err != nil {
		logrus.Errorf("%v", err)
		return nil
	}
	go func() {
		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				logrus.Errorf("read message error: %v.\n", err)
				return
			}
			read <- message
			logrus.Errorf("received message: [%s]\n", message)
		}
	}()

	go func() {
		for {
			select {
			case message := <-write:
				err := conn.WriteMessage(websocket.BinaryMessage, message)
				if err != nil {
					logrus.Errorf("data [%s] is writed to chan error: %s", message, err)
					return
				}
			}
		}
	}()

	return conn
}

func CloseConn(conn *websocket.Conn) {

}
