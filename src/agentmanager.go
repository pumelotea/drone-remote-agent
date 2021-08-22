package main

import (
	"encoding/json"
	"fmt"
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
	for k, v := range manager.AgentHandlerPool {
		fmt.Println(k, v)
		list = append(list, &DashboardDataItem{
			Id:       v.Id,
			Mode:     v.Mode,
			RDataLen: v.RDataLen,
			SDataLen: v.SDataLen,
			IP:       v.IP,
		})
	}
	return json.Marshal(list)
}
