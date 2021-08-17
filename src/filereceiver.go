package main

import (
	"fmt"
	"github.com/pkg/sftp"
	"github.com/tidwall/gjson"
	"golang.org/x/crypto/ssh"
	"log"
)

type FileReceiver struct {
	Agent        *Agent
	AgentHandler *AgentHandler
	Done         chan struct{}
	SSHHost      string
	SSHUsername  string
	SSHPassword  string
	FilePath     string
	FileLength   int64
}

func NewFileReceiver(agent *Agent, agentHandler *AgentHandler, params string) *FileReceiver {
	sshHost := gjson.Get(params, "payload.sshHost").String()
	sshUsername := gjson.Get(params, "payload.sshUsername").String()
	sshPassword := gjson.Get(params, "payload.sshPassword").String()
	filePath := gjson.Get(params, "payload.filePath").String()
	fileLength := gjson.Get(params, "payload.fileLength").Int()

	return &FileReceiver{
		Agent:        agent,
		AgentHandler: agentHandler,
		Done:         make(chan struct{}),
		SSHHost:      sshHost,
		SSHUsername:  sshUsername,
		SSHPassword:  sshPassword,
		FilePath:     filePath,
		FileLength:   fileLength,
	}
}

func (receiver *FileReceiver) Handle() {
	defer close(receiver.Done)
	// 把2个管道进行对接
	dstFile, sshClient, sftpClient, err := receiver.sftpOverWs()
	if err != nil {
		fmt.Println("[Agent][FileReceiver][Handle]", err)
		return
	}
	defer dstFile.Close()
	defer sshClient.Close()
	defer sftpClient.Close()

	var sendLen int64 = 0
	//开启数据读取推送循环
	for {
		// 文件接收器的数据一定为二进制数据块
		data, err := receiver.AgentHandler.read()
		if err != nil {
			log.Println("[Agent][FileReceiver]", err)
			break
		}
		log.Println("[Agent][FileReceiver] Data Len =", len(data))
		n, err := dstFile.Write(data)
		sendLen += int64(n)
		if sendLen == receiver.FileLength {
			err = receiver.responseFileUploadCmd(6)
			break
		} else {
			err = receiver.responseFileUploadCmd(2)
		}

		if err != nil {
			fmt.Println("[Agent][FileReceiver][Handle]", err)
			break
		}
	}
}

func (receiver *FileReceiver) responseFileUploadCmd(status int64) error {
	return receiver.AgentHandler.writeResponse(2, &ResFileUploadCmd{Status: status})
}

func (receiver *FileReceiver) sftpOverWs() (*sftp.File, *ssh.Client, *sftp.Client, error) {
	sshClient, err := receiver.openSSH()
	if err != nil {
		fmt.Println("[Agent][FileReceiver][SftpOverWs]", err)
		return nil, nil, nil, err
	}

	sftpClient, err := receiver.openSFTP(sshClient)
	if err != nil {
		fmt.Println("[Agent][FileReceiver][SftpOverWs]", err)
		return nil, nil, nil, err
	}

	dstFile, err := sftpClient.Create(receiver.FilePath)
	if err != nil {
		fmt.Println("[Agent][FileReceiver][SftpOverWs]", err)
		return nil, nil, nil, err
	}
	return dstFile, sshClient, sftpClient, err
}

func (receiver *FileReceiver) openSSH() (*ssh.Client, error) {
	con := &ssh.ClientConfig{
		User:            receiver.SSHUsername,
		Auth:            []ssh.AuthMethod{ssh.Password(receiver.SSHPassword)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	client, err := ssh.Dial("tcp", receiver.SSHHost, con)
	if err != nil {
		fmt.Println("[Agent][FileReceiver][OpenSSH]", err)
		return nil, err
	}

	return client, nil
}

func (receiver *FileReceiver) openSFTP(sshClient *ssh.Client) (*sftp.Client, error) {
	sftpClient, err := sftp.NewClient(sshClient)
	if err != nil {
		fmt.Println("[Agent][FileReceiver][OpenSFTP]", err)
		return nil, err
	}
	return sftpClient, nil
}
