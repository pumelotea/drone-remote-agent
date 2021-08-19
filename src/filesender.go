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
	Client                           *Client
	ClientHandler                    *ClientHandler
	Done                             chan struct{}
	FilePath                         string
	FileRemotePath                   string
	FileDataPackQueue                []*FileBlock
	FileDataPackQueueContinue        chan struct{}
	FileDataPackQueueContinueBlocked bool
	FileSendContinue                 chan struct{}
	FileSendContinueBlocked          bool
	FileDataPackQueueMaxLen          int
	FileLength                       int64
	UpLength                         int64
}

func NewFileSender(client *Client, clientHandler *ClientHandler, filePath string, fileRemotePath string) *FileSender {
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		log.Fatalln("[Client][NewFileSender]", err)
	}
	return &FileSender{
		Client:                           client,
		ClientHandler:                    clientHandler,
		FilePath:                         filePath,
		FileRemotePath:                   fileRemotePath,
		FileLength:                       fileInfo.Size(),
		Done:                             make(chan struct{}),
		FileDataPackQueueContinue:        make(chan struct{}),
		FileDataPackQueueContinueBlocked: false,
		FileSendContinue:                 make(chan struct{}),
		FileDataPackQueue:                make([]*FileBlock, 0),
		FileSendContinueBlocked:          false,
		FileDataPackQueueMaxLen:          10,
		UpLength:                         0,
	}
}

func (sender *FileSender) Handle() {
	defer close(sender.Done)
	go sender.startFileReader()

	// 发送第一帧数据包，响应服务端首次同意指令。
	b := sender.popFileBlock()
	sender.ClientHandler.writeRaw(b)

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
			b := sender.popFileBlock()
			sender.ClientHandler.writeRaw(b)

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

func (sender *FileSender) startFileReader() {
	reader, file := sender.fileOverWs()
	defer file.Close()
	for {
		//判断队列是否已经存满
		if len(sender.FileDataPackQueue) >= sender.FileDataPackQueueMaxLen {
			//阻塞读取
			sender.FileDataPackQueueContinueBlocked = true
			<-sender.FileDataPackQueueContinue
		}

		buf := make([]byte, 102400)
		n, err := reader.Read(buf)
		if err == io.EOF {
			close(sender.FileDataPackQueueContinue)
			return
		}

		eData, err := sender.Client.encode(buf[:n])
		if err != nil {
			log.Println("[Client][Encode]", err)
			return
		}

		// 加入文件队列
		sender.FileDataPackQueue = append(sender.FileDataPackQueue, &FileBlock{
			RawLen:     n,
			EnCodeData: eData,
		})

		// 如果发送阻塞
		if sender.FileSendContinueBlocked {
			// 释放阻塞
			sender.FileSendContinueBlocked = false
			sender.FileSendContinue <- struct{}{}
		}
	}
}

func (sender *FileSender) popFileBlock() []byte {
	if len(sender.FileDataPackQueue) == 0 {
		sender.FileSendContinueBlocked = true
		<-sender.FileSendContinue
	}
	data := sender.FileDataPackQueue[0]
	sender.FileDataPackQueue = sender.FileDataPackQueue[1:]

	sender.UpLength += int64(data.RawLen)
	sender.printPercent()

	// 如果文件读取阻塞
	if sender.FileDataPackQueueContinueBlocked {
		sender.FileDataPackQueueContinueBlocked = false
		sender.FileDataPackQueueContinue <- struct{}{}
	}

	return data.EnCodeData
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
