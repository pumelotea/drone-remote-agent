package main

import (
	"encoding/json"
)

type AgentManager struct {
	AgentHandlerPool map[string]*AgentHandler
}

func NewAgentManager() *AgentManager {
	return &AgentManager{
		AgentHandlerPool: make(map[string]*AgentHandler),
	}
}

func (manager *AgentManager) Register(handler *AgentHandler) {
	manager.AgentHandlerPool[handler.Id] = handler
}

func (manager *AgentManager) UnRegister(handler *AgentHandler) {
	delete(manager.AgentHandlerPool, handler.Id)
}

func (manager *AgentManager) JSON() ([]byte, error) {
	list := make([]*DashboardDataItem, 0)
	for _, v := range manager.AgentHandlerPool {
		fileList := make([]*DashboardDataFileItem, 0)
		for i := 0; i < len(v.FileReceiverList); i++ {
			fileList = append(fileList, &DashboardDataFileItem{
				SSHHost:       v.FileReceiverList[i].SSHHost,
				SSHUsername:   v.FileReceiverList[i].SSHUsername,
				FilePath:      v.FileReceiverList[i].FilePath,
				FileLength:    v.FileReceiverList[i].FileLength,
				ReceiveLength: v.FileReceiverList[i].ReceiveLength,
			})
		}

		list = append(list, &DashboardDataItem{
			Id:        v.Id,
			Mode:      v.Mode,
			RDataLen:  v.RDataLen,
			SDataLen:  v.SDataLen,
			IP:        v.IP,
			FileList:  fileList,
			CreatedAt: v.CreatedAt,
		})
	}
	return json.Marshal(list)
}
