package main

type ReqData struct {
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
