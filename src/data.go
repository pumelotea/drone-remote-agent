package main

type PackData struct {
	Len  int
	Pwd  []byte
	Data []byte
}

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
	FilePath    string `json:"filePath"`
	FileLength  int64  `json:"fileLength"`
}

type ResFileUploadCmd struct {
	// 0请求发送，1开始，2继续发送下个分片，3超时，4错误，5重传，6结束
	Status int64 `json:"status"`
}

type ReqHandShakeCmd struct {
	Password string `json:"password"`
}

type ResHandShakeCmd struct {
	// 0 失败，1通过
	Status int64 `json:"status"`
}

type FileBlock struct {
	RawLen     int
	EnCodeData []byte
}

type DashboardDataItem struct {
	Id       string
	Mode     int64
	RDataLen int64
	SDataLen int64
	IP       string
	FileList []*DashboardDataFileItem
}

type DashboardDataFileItem struct {
	SSHHost       string
	SSHUsername   string
	FilePath      string
	FileLength    int64
	ReceiveLength int64
}
