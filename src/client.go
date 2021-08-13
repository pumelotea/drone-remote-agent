package main

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/tidwall/gjson"
	"github.com/wenzhenxi/gorsa"
	"log"
	"net/url"
	"os"
	"strings"
)

type Client struct {
	PublicKey         string
	PublicKeyFilePath string
	AgentEndpoint     string
	SSHHost           string
	SSHUsername       string
	SSHPassword       string
	Scripts           string
}

func NewClient(publicKeyFilePath string) *Client {
	var client = &Client{
		PublicKeyFilePath: publicKeyFilePath,
	}
	// 加载证书-公钥
	err := client.loadPublicKey()
	if err != nil {
		log.Fatalln("[Client][Load PublicKey]", err)
	}
	return client
}

func (client *Client) wsWrite(conn *websocket.Conn, cmd int64, reqCmd *ReqCmd) error {
	reqData := ReqData{
		Cmd:     cmd,
		Payload: reqCmd,
	}

	b, err := json.Marshal(reqData)
	if err != nil {
		log.Fatalln("[Client][JSON Marshal]", err)
		return err
	}

	eData, err := client.encode(b)
	if err != nil {
		log.Fatalln("[Client][Encode]", err)
		return err
	}

	err = conn.WriteMessage(websocket.TextMessage, []byte(eData))
	if err != nil {
		log.Fatalln("[Client][WS Write]", err)
		return err
	}
	return nil
}

func (client *Client) loadPublicKey() error {
	b, err := ReadAll(client.PublicKeyFilePath)
	if err != nil {
		log.Println("[Client][ReadFile]", err)
		return err
	}
	client.PublicKey = string(b)
	return nil
}

func (client *Client) Connect() {
	u := url.URL{Scheme: "ws", Host: client.AgentEndpoint, Path: "/agent"}
	log.Printf("[Client] connecting to %s \n", u.String())

	conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Println("[Client][WS Dial]", err)
		os.Exit(100)
	}
	defer conn.Close()

	done := make(chan struct{})

	go func() {
		defer close(done)
		fmt.Println("==================execute logs==================")
		for {
			_, data, err := conn.ReadMessage()
			if err != nil {
				log.Println("[Client][WS ReadMessage]", err)
				return
			}

			dData, err := client.decode(data)
			if err != nil {
				log.Println("[Client][Decode]", err)
				return
			}

			cmd := gjson.Get(dData, "cmd").Int()
			content := gjson.Get(dData, "payload.content").String()
			exitCode := gjson.Get(dData, "payload.exitCode").Int()

			resCmd := &ResCmd{
				Content:  content,
				ExitCode: exitCode,
			}

			// 主动断开
			if cmd == 200 {
				os.Exit(0)
			}

			fmt.Printf(resCmd.Content)

			if resCmd.ExitCode != 0 {
				os.Exit(int(resCmd.ExitCode))
			}
		}
	}()

	// 输出以下将要执行的脚本

	scriptArr := strings.Split(client.Scripts,";")

	fmt.Println("=====================script=====================")
	for i := 0; i < len(scriptArr); i++ {
		fmt.Println(scriptArr[i])
	}


	// 发送消息
	err = client.wsWrite(conn, 1, &ReqCmd{
		SSHHost:     client.SSHHost,
		SSHUsername: client.SSHUsername,
		SSHPassword: client.SSHPassword,
		Scripts:     client.Scripts,
	})

	if err != nil {
		log.Fatalln("[Client][WS Write]", err)
	}

	<-done
}

func (client *Client) decode(raw []byte) (string, error) {
	dData, err := gorsa.PublicDecrypt(string(raw), client.PublicKey)
	if err != nil {
		log.Println("[Client][Decode]", err)
		return "", err
	}
	return dData, nil
}

func (client *Client) encode(raw []byte) (string, error) {
	eData, err := gorsa.PublicEncrypt(string(raw), client.PublicKey)
	if err != nil {
		log.Println("[Client][Encode]", err)
		return "", err
	}
	return eData, nil
}
