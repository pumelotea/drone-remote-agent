package main

type ReqData struct {
	//0指令请求，1脚本请求，2文件请求
	Cmd     int64       `json:"cmd"`
	Payload interface{} `json:"payload"`
}

type ResData struct {
	Cmd     int64       `json:"cmd"`
	Payload interface{} `json:"payload"`
}

type ReqCmd struct {
	SSHHost     string `json:"sshHost"`
	SSHUsername string `json:"sshUsername"`
	SSHPassword string `json:"sshPassword"`
	Scripts     string `json:"scripts"`
}

type ResCmd struct {
	Content  string `json:"content"`
	ExitCode int64  `json:"exitCode"`
}

type ReqFileUploadCmd struct {
	SSHHost     string `json:"sshHost"`
	SSHUsername string `json:"sshUsername"`
	SSHPassword string `json:"sshPassword"`
	FileLength  int64  `json:"fileLength"`
}

type ResFileUploadCmd struct {
	// 0请求发送，1继续发送下个分片，2超时，3错误，4重传，5，结束
	Status int64 `json:"status"`
}

type ReqHandShakeCmd struct {
	Password string `json:"password"`
}

type ResHandShakeCmd struct {
	// 0 失败，1通过
	Status int64 `json:"status"`
}
