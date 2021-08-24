package main

import (
	"encoding/json"
	"fmt"
	"github.com/go-basic/uuid"
	"github.com/gorilla/websocket"
	"github.com/tidwall/gjson"
	"golang.org/x/crypto/ssh"
	"log"
	"time"
)

type AgentHandler struct {
	Id               string
	Agent            *Agent
	Conn             *websocket.Conn
	Mode             int64 //0普通模式，1文件传输模式
	RDataLen         int64 // 统计数据
	SDataLen         int64
	IP               string
	FileReceiverList []*FileReceiver
	CreatedAt        int64
}

func NewAgentHandler(agent *Agent, conn *websocket.Conn) *AgentHandler {
	handler := &AgentHandler{
		Id:        uuid.New(),
		Agent:     agent,
		Conn:      conn,
		Mode:      0,
		RDataLen:  0,
		SDataLen:  0,
		IP:        conn.RemoteAddr().String(),
		CreatedAt: time.Now().Unix(),
	}
	agent.Manager.Register(handler)
	return handler
}

func (handler *AgentHandler) Handle() {
	defer handler.Agent.Manager.UnRegister(handler)
	for {
		data, err := handler.read()
		if err != nil {
			log.Println("[Agent][Handle]", err)
			goto End
		}
		// 处理器收到的数据一定解码后一定是文本json
		dDataString := string(data)
		log.Println("[Agent][Handle]", dDataString)
		cmd := gjson.Get(dDataString, "cmd").Int()
		switch cmd {
		case 0:
			//握手请求
			err = handler.handshake(dDataString)
			if err != nil {
				log.Println("[Agent][HandShake]", err)
				goto End
			}
		case 1:
			//脚本请求
			err = handler.execute(dDataString)
			if err != nil {
				log.Println("[Agent][Execute Over SSH]", err)
				goto End
			}
		case 2:
			//文件请求
			//创建文件接收器
			fileReceiver := NewFileReceiver(handler.Agent, handler, dDataString)
			handler.FileReceiverList = append(handler.FileReceiverList, fileReceiver)
			go fileReceiver.Handle()
			//响应开始传输
			err = fileReceiver.responseFileUploadCmd(1)
			if err != nil {
				fmt.Println("[Agent][FileReceiver][Handle]", err)
				goto End
			}
			//切换模式为文件传输
			handler.Mode = 1

			//阻塞大循环
			<-fileReceiver.Done

			//释放后还原为
			handler.Mode = 0
		}

	}
End:
	log.Println("[Agent]", "WebSocket Closed")
}

func (handler *AgentHandler) handshake(dataString string) error {
	password := gjson.Get(dataString, "payload.password").String()
	if password != "drone-remote-agent" {
		return handler.writeResponse(0, &ResHandShakeCmd{Status: 0})
	}
	return handler.writeResponse(0, &ResHandShakeCmd{Status: 1})
}

func (handler *AgentHandler) execute(dataString string) error {
	sshHost := gjson.Get(dataString, "payload.sshHost").String()
	sshUsername := gjson.Get(dataString, "payload.sshUsername").String()
	sshPassword := gjson.Get(dataString, "payload.sshPassword").String()
	scripts := gjson.Get(dataString, "payload.scripts").String()
	reqCmd := &ReqCmd{
		SSHHost:     sshHost,
		SSHUsername: sshUsername,
		SSHPassword: sshPassword,
		Scripts:     scripts,
	}
	//执行脚本
	client, err := ssh.Dial("tcp", reqCmd.SSHHost, &ssh.ClientConfig{
		User:            reqCmd.SSHUsername,
		Auth:            []ssh.AuthMethod{ssh.Password(reqCmd.SSHPassword)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	})

	if err != nil {
		log.Println("[Agent][SSH Dial]", err)
		err := handler.writeResponse(1, &ResCmd{
			Content:  err.Error(),
			ExitCode: 1,
		})
		if err != nil {
			log.Println("[Agent][WS Write]", err)
			return err
		}
		return err
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		log.Println("[Agent][SSH NewSession]", err)
		err := handler.writeResponse(1, &ResCmd{
			Content:  err.Error(),
			ExitCode: 2,
		})
		if err != nil {
			log.Println("[Agent][WS Write]", err)
			return err
		}
		return err
	}
	defer session.Close()

	//执行脚本
	bs, err := session.CombinedOutput(reqCmd.Scripts)
	if err != nil {
		log.Println("[Agent][SSH CombinedOutput]", string(bs))
		log.Println("[Agent][SSH CombinedOutput]", err)
		err := handler.writeResponse(1, &ResCmd{
			Content:  string(bs),
			ExitCode: 3,
		})
		if err != nil {
			log.Println("[Agent][WS Write]", err)
			return err
		}
		return err
	}

	// 返回执行结果
	err = handler.writeResponse(1, &ResCmd{
		Content:  string(bs),
		ExitCode: 0,
	})
	if err != nil {
		log.Println("[Agent][WS Write]", err)
		return err
	}

	// 结束通知
	err = handler.writeResponse(1, &ResCmd{
		Content:  "✅ Successfully executed commands to all host.[SENT FROM DRA AGENT]\n",
		ExitCode: 0,
	})
	if err != nil {
		log.Println("[Agent][WS Write]", err)
		return err
	}

	// 200结束指令，通知客户端断开连接
	err = handler.writeResponse(200, &ResCmd{
		Content:  "",
		ExitCode: 0,
	})

	if err != nil {
		log.Println("[Agent][WS Write]", err)
		return err
	}
	return nil

}

func (handler *AgentHandler) read() ([]byte, error) {
	mt, data, err := handler.Conn.ReadMessage()
	// 接收流量计数器
	handler.RDataLen += int64(len(data))
	if mt != websocket.BinaryMessage {
		log.Println("[Agent][WS ReadMessage]", "MsgType Not BinaryMessage")
		return nil, err
	}
	if err != nil {
		log.Println("[Agent][WS ReadMessage]", err)
		return nil, err
	}
	//log.Println("[Agent][WS Message Raw Byte Len]", len(data))

	dData, err := handler.Agent.decode(data)
	if err != nil {
		log.Println("[Agent][Decode]", err)
		return nil, err
	}
	return dData, nil
}

func (handler *AgentHandler) writeByte(data []byte) error {
	eData, err := handler.Agent.encode(data)
	if err != nil {
		log.Println("[Agent][Encode]", err)
		return err
	}
	// 发送流量计数器
	handler.SDataLen += int64(len(eData))
	return handler.Conn.WriteMessage(websocket.BinaryMessage, eData)
}

func (handler *AgentHandler) writeResponse(cmd int64, resData interface{}) error {
	res := ResData{
		Cmd:     cmd,
		Payload: resData,
	}
	b, err := json.Marshal(res)
	if err != nil {
		log.Println("[Agent][JSON Marshal]", err)
		return err
	}

	return handler.writeByte(b)
}
