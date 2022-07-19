package main

import (
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/wenzhenxi/gorsa"
	"log"
	"net/http"
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
	RSA               *gorsa.RSASecurity
	FileList          []string
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

	client.RSA = &gorsa.RSASecurity{}
	err = client.RSA.SetPublicKey(client.PublicKey)
	if err != nil {
		log.Fatalln("[Agent][RSA SetPublicKey]", err)
	}
	return client
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

	header := http.Header{}
	header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/103.0.5060.114 Safari/537.36 Edg/103.0.1264.62")
	conn, _, err := websocket.DefaultDialer.Dial(u.String(), header)
	if err != nil {
		log.Println("[Client][WS Dial]", err)
		os.Exit(100)
	}
	defer conn.Close()

	clientHandler := NewClientHandler(client, conn)
	go clientHandler.Handle()

	// 输出以下将要执行的脚本
	scriptArr := strings.Split(client.Scripts, ";")

	for i := 0; i < len(scriptArr); i++ {
		fmt.Println(scriptArr[i])
	}

	err = clientHandler.reqHandShack()
	if err != nil {
		log.Fatalln("[Client][ReqHandShack]", err)
	}

	<-clientHandler.Done
}

func (client *Client) decode(raw []byte) ([]byte, error) {
	pack, err := UnPackBytes(raw)
	if err != nil {
		log.Println("[Client][UnPackBytes]", err)
		return nil, err
	}

	dPwd, err := client.RSA.PubKeyDECRYPT(pack.Pwd)
	if err != nil {
		log.Println("[Client][PubKeyDECRYPT]", err)
		return nil, err
	}

	dData, err := AesDeCrypt(pack.Data, dPwd)
	if err != nil {
		log.Println("[Client][AesDeCrypt]", err)
		return nil, err
	}

	return dData, nil
}

func (client *Client) encode(raw []byte) ([]byte, error) {
	pwd := GenerateAESPwd()
	pData, err := AesEcrypt(raw, pwd)
	if err != nil {
		log.Println("[Client][AesEcrypt]", err)
		return nil, err
	}
	ePwd, err := client.RSA.PubKeyENCTYPT(pwd)
	if err != nil {
		log.Println("[Client][PubKeyENCTYPT]", err)
		return nil, err
	}

	eData := PackBytes(ePwd, pData)
	return eData, nil
}
