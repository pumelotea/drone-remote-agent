# drone-remote-agent 

> 简称DRA

### Agent 模块
> 主要负责启动一个websocket server
### Client 模块
> 主要负责启动一个websocket client


## 编译

### 常规编译
```shell
go build main.go agent.go client.go data.go util.go
```

### Docker编译封装
```shell
docker build -t dra:1.0 .
```

## 秘钥公钥生成
```shell
openssl genrsa -out rsa_private_key.pem 1024
openssl rsa -in rsa_private_key.pem -pubout -out rsa_public_key.pem
```

## 运行
Agent启动
```shell
./main --mode agent --prk /path/rsa_private_key.pem
```
默认Agent监听在`0.0.0.0:8080`，可以通过参数`--listen 127.0.0.1:8080`

Client启动

```SHELL
PLUGIN_AGENT-ENDPOINT=127.0.0.1:8080;PLUGIN_SSH-HOST=10.10.0.27:22;PLUGIN_SSH-USERNAME=root;PLUGIN_SSH-PASSWORD=123456;PLUGIN_PUBLICKEYFILEPATH=/path/rsa_public_key.pem;PLUGIN_SCRIPT=ls ./main
```

Drone中的配置
```YML
kind: pipeline
name: default
type: docker

trigger:
  branch:
    - master
  event:
    - push

volumes:
  - name: wsKey
    host:
      path: /data/dra/

steps:
  - name: deploy-container
    pull: if-not-exists
    image: pumelo/dra:1.0 #官方仓库有现有镜像
    volumes:
      - name: wsKey
        path: /dra
    settings:
      agent-endpoint: 10.10.0.27:8080
      ssh-host: 10.10.0.27:22
      ssh-username: root
      ssh-password: 123456
      publicKeyFilePath: /dra/rsa_public_key.pem
      script:
        - docker pull 10.10.0.14:5000/nginx:1.15
        - docker run -d \
        - --name=test-nginx-a \
        - -p8877:80 \
        - 10.10.0.14:5000/nginx:1.15
```

## 参数表

命令行参支持
```
--mode agent/client, default is client
--prk privateKeyFilePath, like /path/foo [ONLY AGENT]
--pbk publicKeyFilePath, like /path/foo [ONLY CLIENT]
--endpoint, like 127.0.0.1:8080 [ONLY AGENT]
--listen, like 0.0.0.0:8080 [ONLY CLIENT]
--sshHost, like 0.0.0.0:8080 [ONLY CLIENT]
--sshUsername, like root [ONLY CLIENT]
--sshPassword, like 123456 [ONLY CLIENT]
--script, like ls [ONLY CLIENT]
```

环境变量支持,与命令行参数一一对应
```
PLUGIN_MODE
PLUGIN_PRIVATEKEYFILEPATH
PLUGIN_PUBLICKEYFILEPATH
PLUGIN_AGENT-ENDPOINT
PLUGIN_LISTENADDR
PLUGIN_SSH-HOST
PLUGIN_SSH-USERNAME
PLUGIN_SSH-PASSWORD
PLUGIN_SCRIPT
```

如果同时使用2种传参方式，那么命令行参数优先于环境变量，空值会忽略。