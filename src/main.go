package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
)

var (
	gitHash   string
	buildTime string
	goVersion string
	version   string
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
var uploads string
var listenAddr = "0.0.0.0:8080"
var useSSL = false

func init() {
	parseVar()
}

func parseVar() {
	args := os.Args
	if len(args) == 2 && (args[1] == "--version" || args[1] == "-v") {
		fmt.Printf("Welcome to Use Drone Remote Agent.\n")
		fmt.Printf("Version: %s \n", version)
		fmt.Printf("Git Commit Hash: %s \n", gitHash)
		fmt.Printf("Build TimeStamp: %s \n", buildTime)
		fmt.Printf("GoLang Version: %s \n", goVersion)
		mode = "version"
		return
	}
	parseEnvVar()
	_mode := flag.String("mode", "client", "--mode agent/client, default is client")
	_prk := flag.String("prk", "", "--prk privateKeyFilePath, like /path/foo")
	_pbk := flag.String("pbk", "", "--pbk publicKeyFilePath, like /path/foo")
	_endpoint := flag.String("endpoint", "", "--endpoint, like 127.0.0.1:8080")
	_useSSL := flag.String("useSSL", "", "--useSSL, like true")
	_listen := flag.String("listen", "0.0.0.0:8080", "--listen, like 0.0.0.0:8080")
	_sshHost := flag.String("sshHost", "", "--sshHost, like 0.0.0.0:8080")
	_sshUsername := flag.String("sshUsername", "", "--sshUsername, like root")
	_sshPassword := flag.String("sshPassword", "", "--sshPassword, like 123456")
	_script := flag.String("script", "", "--script, like ls")
	_upload := flag.String("upload", "", "--upload, like /local/path/file1:/remote/path/file1,...")
	flag.Parse()

	mode = NotEmptyCopy(mode, *_mode)
	privateKeyFilePath = NotEmptyCopy(privateKeyFilePath, *_prk)
	publicKeyFilePath = NotEmptyCopy(publicKeyFilePath, *_pbk)
	agentEndpoint = NotEmptyCopy(agentEndpoint, *_endpoint)
	if *_useSSL != "" {
		useSSL = NotEmptyCopy(listenAddr, *_useSSL) == "true"
	}
	listenAddr = NotEmptyCopy(listenAddr, *_listen)
	sshHost = NotEmptyCopy(sshHost, *_sshHost)
	sshUsername = NotEmptyCopy(sshUsername, *_sshUsername)
	sshPassword = NotEmptyCopy(sshPassword, *_sshPassword)
	scripts = NotEmptyCopy(scripts, *_script)
	scripts = CombineScriptIntoOneLine(scripts)
	uploads = NotEmptyCopy(uploads, *_upload)

}

func parseEnvVar() {
	mode = os.Getenv("PLUGIN_MODE")
	privateKeyFilePath = os.Getenv("PLUGIN_PRIVATEKEYFILEPATH")
	publicKeyFilePath = os.Getenv("PLUGIN_PUBLICKEYFILEPATH")
	agentEndpoint = os.Getenv("PLUGIN_AGENT-ENDPOINT")
	useSSL = os.Getenv("PLUGIN_USE-SSL") == "true"
	listenAddr = os.Getenv("PLUGIN_LISTENADDR")
	sshHost = os.Getenv("PLUGIN_SSH-HOST")
	sshUsername = os.Getenv("PLUGIN_SSH-USERNAME")
	sshPassword = os.Getenv("PLUGIN_SSH-PASSWORD")
	scripts = os.Getenv("PLUGIN_SCRIPT")
	uploads = os.Getenv("PLUGIN_UPLOAD")
	scripts = strings.ReplaceAll(scripts, ",", ";")
}

func main() {
	switch mode {
	case "client":
		client := NewClient(publicKeyFilePath)
		client.AgentEndpoint = agentEndpoint
		client.UseSSL = useSSL
		client.SSHHost = sshHost
		client.SSHUsername = sshUsername
		client.SSHPassword = sshPassword
		client.Scripts = scripts
		if len(uploads) > 0 {
			client.FileList = strings.Split(uploads, ",")
		}
		client.Connect()
	case "version":
	default:
		agent := NewAgent(listenAddr, privateKeyFilePath)
		err := agent.Serve()
		if err != nil {
			log.Fatalln("[Agent][Serve]", err)
		}
	}
}
