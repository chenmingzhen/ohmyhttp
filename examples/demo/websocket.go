package demo

import (
	"net/http"

	"github.com/gorilla/websocket"
)

const maxWebSocketMessageBytes = 64 << 10

var websocketUpgrader = websocket.Upgrader{}

// WebSocketEcho 演示 HandleHTTP 如何把 ResponseWriter 交给 WebSocket 库完成
// HTTP Upgrade。连接建立后，服务端会原样返回客户端发送的文本或二进制消息。
func (*Handler) WebSocketEcho(writer http.ResponseWriter, request *http.Request) {
	connection, err := websocketUpgrader.Upgrade(writer, request, nil)
	if err != nil {
		return
	}
	defer connection.Close()

	connection.SetReadLimit(maxWebSocketMessageBytes)
	for {
		messageType, message, err := connection.ReadMessage()
		if err != nil {
			return
		}
		if err := connection.WriteMessage(messageType, message); err != nil {
			return
		}
	}
}
