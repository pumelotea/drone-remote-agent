package main

import (
	"encoding/json"
	"github.com/gorilla/websocket"
	"github.com/tidwall/gjson"
	"github.com/wenzhenxi/gorsa"
	"golang.org/x/crypto/ssh"
	"log"
	"net/http"
)

type Agent struct {
	Addr               string
	PrivateKey         string
	PrivateKeyFilePath string
	wsUpGrader         websocket.Upgrader
}

func NewAgent(addr string, privateKeyFilePath string) *Agent {
	var upGrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	} // use default options

	var agent = &Agent{
		Addr:               addr,
		PrivateKey:         "",
		PrivateKeyFilePath: privateKeyFilePath,
		wsUpGrader:         upGrader,
	}

	// 加载证书-私钥
	err := agent.loadPrivateKey()
	if err != nil {
		log.Fatalln("[Agent][Load PrivateKey]", err)
	}
	return agent
}

func (agent *Agent) Serve() error {
	http.HandleFunc("/agent", agent.wsHandle)
	return http.ListenAndServe(agent.Addr, nil)
}

func (agent *Agent) wsHandle(w http.ResponseWriter, r *http.Request) {
	conn, err := agent.wsUpGrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("[Agent][Upgrade WebSocket]", err)
		return
	}

	go func() {
		defer conn.Close()
		for {
			_, data, err := conn.ReadMessage()
			if err != nil {
				log.Println("[Agent][WS ReadMessage]", err)
				break
			}
			log.Println("[Agent][WS Message Raw]", string(data))

			dData, err := agent.decode(data)
			if err != nil {
				log.Println("[Agent][Decode]", err)
				break
			}
			log.Println("[Agent][WS Message Decoded]", dData) //dData 是一个普通string json

			cmd := gjson.Get(dData, "cmd").Int()
			sshHost := gjson.Get(dData, "payload.sshHost").String()
			sshUsername := gjson.Get(dData, "payload.sshUsername").String()
			sshPassword := gjson.Get(dData, "payload.sshPassword").String()
			scripts := gjson.Get(dData, "payload.scripts").String()
			reqCmd := &ReqCmd{
				SSHHost:     sshHost,
				SSHUsername: sshUsername,
				SSHPassword: sshPassword,
				Scripts:     scripts,
			}
			//执行远程命令
			err = agent.executeOverSSH(conn, cmd, reqCmd)
			if err != nil {
				log.Println("[Agent][ExecuteOverSSH]", err)
				break
			}
		}
		log.Println("[Agent]", "Close WebSocket")
	}()
}

func (agent *Agent) loadPrivateKey() error {
	b, err := ReadAll(agent.PrivateKeyFilePath)
	if err != nil {
		log.Println("[Agent][ReadFile]", err)
		return err
	}
	agent.PrivateKey = string(b)
	return nil
}

func (agent *Agent) decode(raw []byte) (string, error) {
	dData, err := gorsa.PriKeyDecrypt(string(raw), agent.PrivateKey)
	if err != nil {
		log.Println("[Agent][Decode]", err)
		return "", err
	}
	return dData, nil
}

func (agent *Agent) encode(raw []byte) (string, error) {
	eData, err := gorsa.PriKeyEncrypt(string(raw), agent.PrivateKey)
	if err != nil {
		log.Println("[Agent][Encode]", err)
		return "", err
	}
	return eData, nil
}

func (agent *Agent) wsWrite(conn *websocket.Conn, cmd int64, resCmd *ResCmd) error {
	res := ResData{
		Cmd:     cmd,
		Payload: resCmd,
	}

	b, err := json.Marshal(res)
	if err != nil {
		log.Println("[Agent][JSON Marshal]", err)
		return err
	}

	eData, err := agent.encode(b)
	if err != nil {
		log.Println("[Agent][Encode]", err)
		return err
	}

	return conn.WriteMessage(websocket.TextMessage, []byte(eData))
}

func (agent *Agent) executeOverSSH(conn *websocket.Conn, cmd int64, reqCmd *ReqCmd) error {
	client, err := ssh.Dial("tcp", reqCmd.SSHHost, &ssh.ClientConfig{
		User:            reqCmd.SSHUsername,
		Auth:            []ssh.AuthMethod{ssh.Password(reqCmd.SSHPassword)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	})
	if err != nil {
		log.Println("[Agent][SSH Dial]", err)
		err := agent.wsWrite(conn, cmd, &ResCmd{
			Content:  err.Error(),
			ExitCode: 1,
		})
		if err != nil {
			log.Println("[Agent][WS Write]", err)
			return err
		}
		return err
	}

	session, err := client.NewSession()
	if err != nil {
		log.Println("[Agent][SSH NewSession]", err)
		err := agent.wsWrite(conn, cmd, &ResCmd{
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
		err := agent.wsWrite(conn, cmd, &ResCmd{
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
	err = agent.wsWrite(conn, cmd, &ResCmd{
		Content:  string(bs),
		ExitCode: 0,
	})
	if err != nil {
		log.Println("[Agent][WS Write]", err)
		return err
	}

	// 结束通知
	err = agent.wsWrite(conn, cmd, &ResCmd{
		Content:  "✅ Successfully executed commands to all host.[SENT FROM DRA AGENT]\n",
		ExitCode: 0,
	})
	if err != nil {
		log.Println("[Agent][WS Write]", err)
		return err
	}

	// 200结束指令，通知客户端断开连接
	err = agent.wsWrite(conn, 200, &ResCmd{
		Content:  "",
		ExitCode: 0,
	})
	if err != nil {
		log.Println("[Agent][WS Write]", err)
		return err
	}
	return nil
}
