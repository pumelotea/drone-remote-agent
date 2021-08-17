package main

import (
	"github.com/gorilla/websocket"
	"github.com/wenzhenxi/gorsa"
	"log"
	"net/http"
)

type Agent struct {
	Addr               string
	PrivateKey         string
	PrivateKeyFilePath string
	wsUpGrader         *websocket.Upgrader
	RSA                *gorsa.RSASecurity
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
		wsUpGrader:         &upGrader,
	}

	// 加载证书-私钥
	err := agent.loadPrivateKey()
	if err != nil {
		log.Fatalln("[Agent][Load PrivateKey]", err)
	}
	agent.RSA = &gorsa.RSASecurity{}
	err = agent.RSA.SetPrivateKey(agent.PrivateKey)
	if err != nil {
		log.Fatalln("[Agent][RSA SetPrivateKey]", err)
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
	agentHandler := NewAgentHandler(agent, conn)
	go agentHandler.Handle()
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

func (agent *Agent) decode(raw []byte) ([]byte, error) {
	dData, err := agent.RSA.PriKeyDECRYPT(raw)
	if err != nil {
		log.Println("[Agent][Decode]", err)
		return nil, err
	}
	return dData, nil
}

func (agent *Agent) encode(raw []byte) ([]byte, error) {
	eData, err := agent.RSA.PriKeyENCTYPT(raw)
	if err != nil {
		log.Println("[Agent][Encode]", err)
		return nil, err
	}
	return eData, nil
}
