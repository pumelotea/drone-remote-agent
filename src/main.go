package main

import (
	"bufio"
	"flag"
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
	"io"
	"log"
	"os"
	"path"
	"strings"
)

// 默认是client模式
var mode = "client"

// 可通过openssl产生
//openssl genrsa -out rsa_private_key.pem 1024
var privateKeyFilePath string

//openssl
//openssl rsa -in rsa_private_key.pem -pubout -out rsa_public_key.pem
var publicKeyFilePath string
var agentEndpoint string
var sshHost string
var sshUsername string
var sshPassword string
var scripts string
var listenAddr = "0.0.0.0:8080"

func init() {
	parseVar()
}

func parseVar() {
	parseEnvVar()
	_mode := flag.String("mode", "client", "--mode agent/client, default is client")
	_prk := flag.String("prk", "", "--prk privateKeyFilePath, like /path/foo")
	_pbk := flag.String("pbk", "", "--pbk publicKeyFilePath, like /path/foo")
	_endpoint := flag.String("endpoint", "", "--endpoint, like 127.0.0.1:8080")
	_listen := flag.String("listen", "0.0.0.0:8080", "--listen, like 0.0.0.0:8080")
	_sshHost := flag.String("sshHost", "", "--sshHost, like 0.0.0.0:8080")
	_sshUsername := flag.String("sshUsername", "", "--sshUsername, like root")
	_sshPassword := flag.String("sshPassword", "", "--sshPassword, like 123456")
	_script := flag.String("script", "", "--script, like ls")
	flag.Parse()

	mode = NotEmptyCopy(mode, *_mode)
	privateKeyFilePath = NotEmptyCopy(privateKeyFilePath, *_prk)
	publicKeyFilePath = NotEmptyCopy(publicKeyFilePath, *_pbk)
	agentEndpoint = NotEmptyCopy(agentEndpoint, *_endpoint)
	listenAddr = NotEmptyCopy(listenAddr, *_listen)
	sshHost = NotEmptyCopy(sshHost, *_sshHost)
	sshUsername = NotEmptyCopy(sshUsername, *_sshUsername)
	sshPassword = NotEmptyCopy(sshPassword, *_sshPassword)
	scripts = NotEmptyCopy(scripts, *_script)

	scripts = CombineScriptIntoOneLine(scripts)

}

func parseEnvVar() {
	mode = os.Getenv("PLUGIN_MODE")
	privateKeyFilePath = os.Getenv("PLUGIN_PRIVATEKEYFILEPATH")
	publicKeyFilePath = os.Getenv("PLUGIN_PUBLICKEYFILEPATH")
	agentEndpoint = os.Getenv("PLUGIN_AGENT-ENDPOINT")
	listenAddr = os.Getenv("PLUGIN_LISTENADDR")
	sshHost = os.Getenv("PLUGIN_SSH-HOST")
	sshUsername = os.Getenv("PLUGIN_SSH-USERNAME")
	sshPassword = os.Getenv("PLUGIN_SSH-PASSWORD")
	scripts = os.Getenv("PLUGIN_SCRIPT")
	scripts = strings.ReplaceAll(scripts, ",", ";")
}

func main() {
	switch mode {
	case "client":
		{
			client := NewClient(publicKeyFilePath)
			client.AgentEndpoint = agentEndpoint
			client.SSHHost = sshHost
			client.SSHUsername = sshUsername
			client.SSHPassword = sshPassword
			client.Scripts = scripts
			client.Connect()
		}
	default:
		{
			agent := NewAgent(listenAddr, privateKeyFilePath)
			err := agent.Serve()
			if err != nil {
				log.Fatalln("[Agent][Serve]", err)
			}
		}
	}
	//test()

}

//------------- TEST
func test() {
	//UploadFile("10.10.0.27", "root", "JSlit0ng+2021_Mong0Db", "/Users/pumelotea/GolandProjects/drone-remote-agent/dra.img", "/root", 22)
	ReadBlock("/Users/pumelotea/GolandProjects/drone-remote-agent/tmp.txt", 128, processBlock)
}

//获取ssh连接
func GetSSHConect(ip, user string, password string, port int) *ssh.Client {
	con := &ssh.ClientConfig{
		User:            user,
		Auth:            []ssh.AuthMethod{ssh.Password(password)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	addr := fmt.Sprintf("%s:%d", ip, port)
	client, err := ssh.Dial("tcp", addr, con)
	if err != nil {
		fmt.Println("Dail failed: ", err)
		panic(err)
	}
	return client
}

//获取ftp连接
func getftpclient(client *ssh.Client) *sftp.Client {
	ftpclient, err := sftp.NewClient(client)
	if err != nil {
		fmt.Println("创建ftp客户端失败", err)
		panic(err)
	}
	return ftpclient
}

//上传文件
func UploadFile(ip, user, password, localpath, remotepath string, port int) {
	client := GetSSHConect(ip, user, password, port)
	ftpclient := getftpclient(client)
	defer ftpclient.Close()

	remoteFileName := path.Base(localpath)
	fmt.Println(localpath, remoteFileName)
	srcFile, err := os.Open(localpath)
	if err != nil {
		fmt.Println("打开文件失败", err)
		panic(err)
	}
	defer srcFile.Close()

	dstFile, e := ftpclient.Create(path.Join(remotepath, remoteFileName))
	if e != nil {
		fmt.Println("创建文件失败", e)
		panic(e)
	}
	defer dstFile.Close()
	buffer := make([]byte, 1024)
	for {
		n, err := srcFile.Read(buffer)
		if err != nil {
			if err == io.EOF {
				fmt.Println("已读取到文件末尾")
				break
			} else {
				fmt.Println("读取文件出错", err)
				panic(err)
			}
		}
		dstFile.Write(buffer[:n])
		//注意，由于文件大小不定，不可直接使用buffer，否则会在文件末尾重复写入，以填充1024的整数倍
	}
	fmt.Println("文件上传成功")
}

//读取文件块
func ReadBlock(filePth string, bufSize int, hookfn func([]byte)) error {
	f, err := os.Open(filePth)
	if err != nil {
		return err
	}
	defer f.Close()

	buf := make([]byte, bufSize) //一次读取多少个字节
	bfRd := bufio.NewReader(f)
	for {
		n, err := bfRd.Read(buf)
		hookfn(buf[:n]) // n 是成功读取字节数

		if err != nil { //遇到任何错误立即返回，并忽略 EOF 错误信息
			if err == io.EOF {
				return nil
			}
			return err
		}
	}
	return nil
}

var xx *websocket.Conn

func processBlock(line []byte) {
	fmt.Print(string(line))
	// 进行加密

	// 发送
	xx.WriteMessage(websocket.BinaryMessage, line)
}
