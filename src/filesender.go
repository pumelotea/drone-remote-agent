package main

import (
	"bufio"
	"github.com/tidwall/gjson"
	"io"
	"log"
	"os"
)

type FileSender struct {
	Client         *Client
	ClientHandler  *ClientHandler
	Done           chan struct{}
	FilePath       string
	FileRemotePath string
	FileLength     int64
}

func NewFileSender(client *Client, clientHandler *ClientHandler, filePath string, fileRemotePath string) *FileSender {
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		log.Fatalln("[Client][NewFileSender]", err)
	}
	return &FileSender{
		Client:         client,
		ClientHandler:  clientHandler,
		FilePath:       filePath,
		FileRemotePath: fileRemotePath,
		FileLength:     fileInfo.Size(),
		Done:           make(chan struct{}),
	}
}

func (sender *FileSender) Handle() {
	defer close(sender.Done)
	reader, file := sender.fileOverWs()
	defer file.Close()
	sender.sendBlock(reader)
	for {
		data, err := sender.ClientHandler.read()
		if err != nil {
			log.Println("[Client][FileSender]", err)
			break
		}
		dataString := string(data)
		status := gjson.Get(dataString, "payload.status").Int()
		switch status {
		case 0:
		case 1:
			//开始发送
			fallthrough
		case 2:
			//继续发送
			sender.sendBlock(reader)
		case 3:
			//超时

		case 4:
			//错误

		case 5:
			//重传

		case 6:
			//结束
			log.Println("[Client][FileSender][Success]", sender.FilePath, "->", sender.FileRemotePath)
			return
		}
	}
}

func (sender *FileSender) requestFileUploadCmd() error {
	return sender.ClientHandler.writeRequest(2, &ReqFileUploadCmd{
		SSHHost:     sender.Client.SSHHost,
		SSHUsername: sender.Client.SSHUsername,
		SSHPassword: sender.Client.SSHPassword,
		FilePath:    sender.FileRemotePath,
		FileLength:  sender.FileLength,
	})
}

func (sender *FileSender) fileOverWs() (*bufio.Reader, *os.File) {
	file, err := os.Open(sender.FilePath)
	if err != nil {
		log.Fatalln("[Client][ReadFile]", err)
	}
	bfRd := bufio.NewReader(file)
	return bfRd, file
}

func (sender *FileSender) sendBlock(reader *bufio.Reader) {
	buf := make([]byte, 128)
	n, err := reader.Read(buf)
	log.Println("[Client][FileSender] send byte len", n)
	if err == io.EOF {
		return
	}
	err = sender.ClientHandler.writeByte(buf[:n])
	if err != nil {
		log.Fatalln("[Client][FileSender]", err)
	}
}
