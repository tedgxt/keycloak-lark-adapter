package api

import (
	"encoding/json"
	"io/ioutil"
	"keycloak-lark-adapter/internal/config"
	log "keycloak-lark-adapter/internal/logger"
	lm "keycloak-lark-adapter/internal/model/lark"
	"net/http"
	"strings"

	"github.com/bitly/go-simplejson"

	"github.com/gin-gonic/gin"

	"github.com/sirupsen/logrus"
)

var (
	logger *logrus.Logger
)

func Init() {
	logger = log.Logger
	SetupRouter()
}

func Healthz(c *gin.Context) {
	c.Status(http.StatusOK)
}

func Notifications(c *gin.Context) {
	data, err := ioutil.ReadAll(c.Request.Body)
	if err != nil {
		logger.Errorf("read request body failed, err: %v", err.Error())
		c.Status(http.StatusInternalServerError)
		return
	}

	logger.Debugf("received lark notification: %v", string(data))

	sj, err := simplejson.NewJson(data)
	if err != nil {
		logger.Errorf("unmarshal lark notification failed, error: %v", err.Error())
		c.Status(http.StatusBadRequest)
		return
	}

	// TODO: encrypt key
	if sj.Get("header").Get("token").MustString() != config.VerificationToken {
		logger.Errorf("token check failed, token: %v", sj.Get("header").Get("token").MustString())
		c.Status(http.StatusBadRequest)
		return
	}

	if sj.Get("type").MustString() == "url_verification" {
		challenge := sj.Get("challenge").MustString()
		m := map[string]string{"challenge": challenge}
		c.JSON(http.StatusOK, m)
		return
	}

	eventType := sj.Get("header").Get("event_type").MustString()
	if strings.Contains(eventType, "contact.user") {
		userMsg := new(lm.ContactUserMsg)
		if err = json.Unmarshal(data, userMsg); err != nil {
			logger.Errorf("failed to parse contact user message, error: %v", err.Error())
			return
		}
		lm.UserChan <- userMsg

		return
	}

	// Process lark department event
	if strings.Contains(eventType, "contact.department") {
		depMsg := new(lm.ContactDepMsg)
		if err = json.Unmarshal(data, depMsg); err != nil {
			logger.Errorf("failed to parse contact department message, error: %v", err.Error())
			return
		}
		lm.DepChan <- depMsg
		return
	}

	c.Status(http.StatusOK)
}
