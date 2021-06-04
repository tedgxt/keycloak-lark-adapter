package main

import (
	"keycloak-lark-adapter/api"
	"keycloak-lark-adapter/cmd/keycloak"
	"keycloak-lark-adapter/cmd/lark"
	"keycloak-lark-adapter/internal/config"
	logger "keycloak-lark-adapter/internal/logger"
	lm "keycloak-lark-adapter/internal/model/lark"
	"keycloak-lark-adapter/pkg/ws"
	"strings"
)

func init() {
	// do not change the init sequence
	config.Init()
	logger.Init()

	ws.Init()
	keycloak.Init()
	lark.Init()
	api.Init()
}

func main() {
	keycloak.ProcessContactEvent(lm.UserChan, lm.DepChan)
	if strings.ToLower(config.EventSource) == "http" {
		r := api.SetupRouter()
		r.Run(":" + config.ServerPort)
		return
	}
	ws.Bot.Run()
}
