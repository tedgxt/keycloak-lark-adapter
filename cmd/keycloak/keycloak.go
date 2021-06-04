package keycloak

import (
	"encoding/json"
	"keycloak-lark-adapter/internal/config"
	"keycloak-lark-adapter/internal/http"
	log "keycloak-lark-adapter/internal/logger"
	"keycloak-lark-adapter/internal/model/keycloak"
	lm "keycloak-lark-adapter/internal/model/lark"
	"keycloak-lark-adapter/pkg/utils"

	"github.com/sirupsen/logrus"
)

const (
	attributePhoneNumber = "phone_number"
	attributeRealName    = "fullname"
	attributeNickname    = "nickname"

	eventTypeUserUpdate       = "contact.user.updated_v3"
	eventTypeUserCreate       = "contact.user.created_v3"
	eventTypeUserDelete       = "contact.user.deleted_v3"
	eventTypeDepartmentUpdate = "contact.department.updated_v3"
	eventTypeDepartmentCreate = "contact.department.created_v3"
	eventTypeDepartmentDelete = "contact.department.deleted_v3"
)

var (
	logger *logrus.Logger
)

func Init() {
	logger = log.Logger
}

func ProcessContactEvent(userChan chan *lm.ContactUserMsg, depChan chan *lm.ContactDepMsg) {
	go func(userChan chan *lm.ContactUserMsg, depChan chan *lm.ContactDepMsg) {
		for {
			select {
			case msg := <-userChan:
				logger.Debugf("preparing to process contact user msg: %v", msg)

				if msg == nil || msg.Header == nil || msg.Header.EventType == "" {
					logger.Errorf("cannot get event type from msg: %#v", msg)
					continue
				}
				err := processUserMsgWithType(msg)
				if err != nil {
					logger.Errorf("process contact msg failed, msg: %#v, error: %v", msg, err)
				}
			case msg := <-depChan:
				logger.Debugf("preparing to process contact department msg: %v", msg)
				if msg == nil || msg.Header == nil || msg.Header.EventType == "" {
					logger.Errorf("cannot get event type from msg: %#v", msg)
					continue
				}
				err := processDepMsgWithType(msg)
				if err != nil {
					logger.Errorf("process contact msg failed, msg: %#v, error: %v", msg, err)
				}
			}
		}
	}(userChan, depChan)
}

func getAppToken() (token string, err error) {
	resp, err := http.Client.R().
		SetHeader("Content-Type", "application/x-www-form-urlencoded").
		SetFormData(map[string]string{"client_id": config.ClientId, "grant_type": "client_credentials", "client_secret": config.ClientSecret}).
		Post(config.Host + "/auth/realms/" + config.Realm + "/protocol/openid-connect/token")
	if err != nil {
		logger.Errorf("get token failed, error: %v", err.Error())
		return
	}
	if !utils.IsSuccessResponse(resp.StatusCode()) {
		logger.Errorf("get token response failed, code: %v, error msg: %v", resp.StatusCode(), string(resp.Body()))
		return
	}
	tokenResp := &keycloak.TokenResp{}
	err = json.Unmarshal(resp.Body(), tokenResp)
	if err != nil {
		logger.Errorf("unmarshal failed, error: %v", err)
		return
	}
	token = "Bearer " + tokenResp.AccessToken
	return
}
