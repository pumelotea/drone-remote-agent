package main

import (
	"bufio"
	"fmt"
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
	UpLength       int64
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
		UpLength:       0,
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
	buf := make([]byte, 102400)
	n, err := reader.Read(buf)
	sender.UpLength += int64(n)
	sender.printPercent()
	//log.Println("[Client][FileSender] sent -> ", sender.UpLength)
	if err == io.EOF {
		return
	}
	err = sender.ClientHandler.writeByte(buf[:n])
	if err != nil {
		log.Fatalln("[Client][FileSender]", err)
	}
}

func (sender *FileSender) printPercent() {
	var size = 50
	var p = float64(sender.UpLength) / float64(sender.FileLength)

	str := fmt.Sprintf("[%s] %.2f%%", bar(int(p*float64(size)), size), p*100)
	fmt.Printf("\r%s", str)
	if sender.UpLength == sender.FileLength {
		fmt.Println(" ✅ ")
	}
}
