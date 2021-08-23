package main

import (
	"github.com/gorilla/websocket"
	"github.com/wenzhenxi/gorsa"
	"log"
	"net/http"
)

type Agent struct {
	Addr               string
	PrivateKey         string
	PrivateKeyFilePath string
	wsUpGrader         *websocket.Upgrader
	RSA                *gorsa.RSASecurity
	Manager            *AgentManager
}

func NewAgent(addr string, privateKeyFilePath string) *Agent {
	var upGrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	} // use default options

	var agent = &Agent{
		Addr:               addr,
		PrivateKey:         "",
		PrivateKeyFilePath: privateKeyFilePath,
		wsUpGrader:         &upGrader,
		Manager:            NewAgentManager(),
	}

	// 加载证书-私钥
	err := agent.loadPrivateKey()
	if err != nil {
		log.Fatalln("[Agent][Load PrivateKey]", err)
	}
	agent.RSA = &gorsa.RSASecurity{}
	err = agent.RSA.SetPrivateKey(agent.PrivateKey)
	if err != nil {
		log.Fatalln("[Agent][RSA SetPrivateKey]", err)
	}
	return agent
}

func (agent *Agent) Serve() error {
	http.HandleFunc("/agent", agent.wsHandle)
	http.HandleFunc("/dashboard", agent.dashHandle)
	http.HandleFunc("/status", agent.dashPageHandle)
	return http.ListenAndServe(agent.Addr, nil)
}

func (agent *Agent) wsHandle(w http.ResponseWriter, r *http.Request) {
	conn, err := agent.wsUpGrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("[Agent][Upgrade WebSocket]", err)
		return
	}
	agentHandler := NewAgentHandler(agent, conn)
	go agentHandler.Handle()
}

func (agent *Agent) loadPrivateKey() error {
	b, err := ReadAll(agent.PrivateKeyFilePath)
	if err != nil {
		log.Println("[Agent][ReadFile]", err)
		return err
	}
	agent.PrivateKey = string(b)
	return nil
}

func (agent *Agent) decode(raw []byte) ([]byte, error) {
	pack, err := UnPackBytes(raw)
	if err != nil {
		log.Println("[Agent][UnPackBytes]", err)
		return nil, err
	}

	dPwd, err := agent.RSA.PriKeyDECRYPT(pack.Pwd)
	if err != nil {
		log.Println("[Agent][PriKeyDECRYPT]", err)
		return nil, err
	}

	dData, err := AesDeCrypt(pack.Data, dPwd)
	if err != nil {
		log.Println("[Agent][AesDeCrypt]", err)
		return nil, err
	}

	return dData, nil
}

func (agent *Agent) encode(raw []byte) ([]byte, error) {
	pwd := GenerateAESPwd()
	pData, err := AesEcrypt(raw, pwd)
	if err != nil {
		log.Println("[Agent][AesEcrypt]", err)
		return nil, err
	}

	ePwd, err := agent.RSA.PriKeyENCTYPT(pwd)
	if err != nil {
		log.Println("[Agent][Encode]", err)
		return nil, err
	}

	eData := PackBytes(ePwd, pData)
	return eData, nil
}

func (agent *Agent) dashHandle(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	data, err := agent.Manager.JSON()
	if err != nil {
		log.Println("[Agent][DashHandle]", err)
	}
	w.Write(data)
}

func (agent *Agent) dashPageHandle(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("<!DOCTYPE html>\n<html lang=\"en\">\n<head>\n  <meta charset=\"UTF-8\">\n  <title>Dra Dashboard</title>\n  <style>\n      body {\n          display: flex;\n          padding: 20px;\n          box-sizing: border-box;\n      }\n\n      #app {\n          max-width: 960px;\n          width: 100%;\n          margin: auto;\n      }\n\n      .item + .item {\n          margin-top: 10px;\n      }\n\n      .flex-center {\n          display: flex;\n          align-items: center\n      }\n\n      .mr-10 {\n          margin-right: 10px\n      }\n\n      .ml-10 {\n          margin-right: 10px\n      }\n\n      .f-b {\n          font-weight: bold;\n      }\n\n      .icon {\n          font-size: 18px !important;\n          font-weight: bold !important;\n      }\n  </style>\n  <link rel=\"stylesheet\" href=\"https://unpkg.com/element-plus/lib/theme-chalk/index.css\">\n</head>\n<body>\n<div id=\"app\">\n  <el-card class=\"item\" shadow=\"never\" v-for=\"e in list\">\n    <div class=\"flex-center\">\n      <div class=\"flex-center\">\n        <div class=\"mr-10\">\n          <el-icon class=\"el-icon-connection icon\"></el-icon>\n        </div>\n        <div class=\"f-b mr-10\">{{e.IP}}</div>\n        <el-tag size=\"small\" v-if=\"e.Mode === 1\">传输模式</el-tag>\n        <el-tag size=\"small\" type=\"warning\" v-else>指令模式</el-tag>\n      </div>\n      <div style=\"flex: 1\"></div>\n      <div class=\"flex-center ml-10\">\n        <div class=\"flex-center mr-10\">\n          <el-icon class=\"el-icon-bottom icon\"></el-icon>\n        </div>\n        <div style=\"font-weight: bold\">{{human(e.RDataLen)}}</div>\n      </div>\n      <div class=\"flex-center ml-10\">\n        <div class=\"flex-center mr-10\">\n          <el-icon class=\"el-icon-top icon\"></el-icon>\n        </div>\n        <div class=\"f-b\">{{human(e.SDataLen)}}</div>\n      </div>\n    </div>\n  </el-card>\n  <el-empty description=\"暂无连接\" v-if=\"list.length === 0\"></el-empty>\n  <el-dialog\n    title=\"连接 Dra Agent\"\n    v-model=\"show\"\n    width=\"30%\"\n    :before-close=\"handleClose\">\n    <el-input v-model=\"addr\" placeholder=\"Agent IP:Port\"></el-input>\n    <template #footer>\n    <span class=\"dialog-footer\">\n      <el-button type=\"primary\" @click=\"connect\">连接</el-button>\n    </span>\n    </template>\n  </el-dialog>\n</div>\n\n<script src=\"https://unpkg.com/vue@next\"></script>\n<script src=\"https://unpkg.com/element-plus/lib/index.full.js\"></script>\n<script>\n    const app = Vue.createApp({\n        data() {\n            return {\n                list: [],\n                addr: \"\",\n                show: false\n            }\n        },\n        methods: {\n            human(size) {\n                if (!size) return \"-\";\n                const fileSize = Number(size)\n                let num = 1024.00;\n                if (fileSize < num)\n                    return fileSize + \"B\";\n                if (fileSize < Math.pow(num, 2))\n                    return (fileSize / num).toFixed(2) + \"KB\";\n                if (fileSize < Math.pow(num, 3))\n                    return (fileSize / Math.pow(num, 2)).toFixed(2) + \"MB\";\n                if (fileSize < Math.pow(num, 4))\n                    return (fileSize / Math.pow(num, 3)).toFixed(2) + \"G\";\n                return (fileSize / Math.pow(num, 4)).toFixed(2) + \"T\";\n            },\n            refresh() {\n                fetch(`//${this.addr}/dashboard`)\n                    .then(response => {\n                        return response.json()\n                    })\n                    .then(data => {\n                        this.list = data\n                    })\n            },\n            connect() {\n                this.show = false\n                setInterval(this.refresh, 1000)\n                localStorage.setItem(\"agent-addr\", this.addr)\n            }\n        },\n        mounted() {\n            console.log(\"page init\")\n            this.addr = localStorage.getItem(\"agent-addr\") || ''\n            if (this.addr) {\n                setInterval(this.refresh, 1000)\n            } else {\n                this.show = true\n            }\n        }\n    })\n    app.use(ElementPlus)\n    app.mount('#app')\n\n</script>\n</body>\n</html>"))
}
