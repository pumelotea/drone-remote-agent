package main

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/tidwall/gjson"
	"log"
	"os"
)

type ClientHandler struct {
	Client *Client
	Conn   *websocket.Conn
	//0普通模式，1文件传输模式
	Mode int64
	Done chan struct{}
}

func NewClientHandler(client *Client, conn *websocket.Conn) *ClientHandler {
	return &ClientHandler{
		Client: client,
		Conn:   conn,
		Mode:   0,
		Done:   make(chan struct{}),
	}
}

func (handler *ClientHandler) Handle() {
	defer close(handler.Done)
	// 处理器收到的数据一定解码后一定是文本json
	for {
		data, err := handler.read()
		if err != nil {
			log.Println("[Client][Handle]", err)
			break
		}
		dDataString := string(data)
		log.Println("[Client][Handle]", dDataString)
		cmd := gjson.Get(dDataString, "cmd").Int()
		switch cmd {
		case 0:
			//握手响应
			success := handler.isHandShackSuccess(dDataString)
			if success {
				// 请求发送文件
				err = handler.reqExecuteCmd()
				if err != nil {
					log.Println("[Agent][ReqExecuteCmd]", err)
				}
			}
		case 1:
			//脚本响应
			handler.processExecuteCmdRes(dDataString)
		case 2:
			//文件请求响应
			//创建文件发送器
			//并且阻塞大循环
		case 200:
			os.Exit(0)
		}
	}
}

func (handler *ClientHandler) read() ([]byte, error) {
	mt, data, err := handler.Conn.ReadMessage()
	if mt != websocket.BinaryMessage {
		log.Println("[Client][WS ReadMessage]", "MsgType Not BinaryMessage")
		return nil, err
	}
	if err != nil {
		log.Println("[Client][WS ReadMessage]", err)
		return nil, err
	}
	log.Println("[Client][WS Message Raw Byte Len]", len(data))

	dData, err := handler.Client.decode(data)
	if err != nil {
		log.Println("[Client][Decode]", err)
		return nil, err
	}
	return dData, nil
}

func (handler *ClientHandler) writeByte(data []byte) error {
	eData, err := handler.Client.encode(data)
	if err != nil {
		log.Println("[Client][Encode]", err)
		return err
	}
	return handler.Conn.WriteMessage(websocket.BinaryMessage, eData)
}

func (handler *ClientHandler) writeRequest(cmd int64, reqData interface{}) error {
	res := ReqData{
		Cmd:     cmd,
		Payload: reqData,
	}
	b, err := json.Marshal(res)
	if err != nil {
		log.Println("[Client][JSON Marshal]", err)
		return err
	}
	return handler.writeByte(b)
}

func (handler *ClientHandler) reqHandShack() error {
	return handler.writeRequest(0, &ReqHandShakeCmd{
		Password: "drone-remote-agent",
	})
}

func (handler *ClientHandler) isHandShackSuccess(dataString string) bool {
	status := gjson.Get(dataString, "payload.status").Int()
	return status == 1
}

func (handler *ClientHandler) reqExecuteCmd() error {
	return handler.writeRequest(1, &ReqCmd{
		SSHHost:     handler.Client.SSHHost,
		SSHUsername: handler.Client.SSHUsername,
		SSHPassword: handler.Client.SSHPassword,
		Scripts:     handler.Client.Scripts,
	})
}

func (handler *ClientHandler) processExecuteCmdRes(dataString string) {
	content := gjson.Get(dataString, "payload.content").String()
	exitCode := gjson.Get(dataString, "payload.exitCode").Int()
	resCmd := &ResCmd{
		Content:  content,
		ExitCode: exitCode,
	}

	fmt.Printf(resCmd.Content)

	if resCmd.ExitCode != 0 {
		os.Exit(int(resCmd.ExitCode))
	}
}
