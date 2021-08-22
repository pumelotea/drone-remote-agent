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
	Manager            *AgentManager
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
		Manager:            NewAgentManager(),
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
	http.HandleFunc("/dashboard", agent.dashHandle)
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
	pack, err := UnPackBytes(raw)
	if err != nil {
		log.Println("[Agent][UnPackBytes]", err)
		return nil, err
	}

	dPwd, err := agent.RSA.PriKeyDECRYPT(pack.Pwd)
	if err != nil {
		log.Println("[Agent][PriKeyDECRYPT]", err)
		return nil, err
	}

	dData, err := AesDeCrypt(pack.Data, dPwd)
	if err != nil {
		log.Println("[Agent][AesDeCrypt]", err)
		return nil, err
	}

	return dData, nil
}

func (agent *Agent) encode(raw []byte) ([]byte, error) {
	pwd := GenerateAESPwd()
	pData, err := AesEcrypt(raw, pwd)
	if err != nil {
		log.Println("[Agent][AesEcrypt]", err)
		return nil, err
	}

	ePwd, err := agent.RSA.PriKeyENCTYPT(pwd)
	if err != nil {
		log.Println("[Agent][Encode]", err)
		return nil, err
	}

	eData := PackBytes(ePwd, pData)
	return eData, nil
}

func (agent *Agent) dashHandle(w http.ResponseWriter, r *http.Request) {
	data, err := agent.Manager.JSON()
	if err != nil {
		log.Println("[Agent][DashHandle]", err)
	}
	w.Write(data)
}
