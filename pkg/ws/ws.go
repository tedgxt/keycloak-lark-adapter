package ws

import (
	"encoding/json"
	"keycloak-lark-adapter/internal/config"
	log "keycloak-lark-adapter/internal/logger"
	lm "keycloak-lark-adapter/internal/model/lark"
	"os"
	"strings"
	"time"

	"github.com/bitly/go-simplejson"
	"github.com/cenkalti/backoff"
	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

var (
	wsAdapterEndpoint string
	logger            *logrus.Logger
	Bot               *LarkGuardBot
)

type LarkGuardBot struct {
	conn *websocket.Conn
}

func Init() {
	logger = log.Logger
	Bot = NewLarkGuardBot()
	Bot.setup()
}

func NewLarkGuardBot() *LarkGuardBot {
	return new(LarkGuardBot)
}

func (l *LarkGuardBot) setup() {
	wsAdapterEndpoint = os.Getenv("WEBSOCKET_ADAPTER_ENDPOINT")
	if len(wsAdapterEndpoint) == 0 {
		logger.Fatalf("cannot get param WEBSOCKET_ADAPTER_ENDPOINT from env")
	}

	l.setupWebsocketConn()
}

func (l *LarkGuardBot) Run() {
	logger.Infof("running with mode websocket")

	// register bot client id
	clientInfo := make(map[string]string)
	clientInfo["app_id"] = config.AppId
	clientInfo["verification_token"] = config.VerificationToken
	clientInfo["encrypt_key"] = config.EncryptKey

	buf, _ := json.Marshal(clientInfo)

	go func() {
		for {
			time.Sleep(5 * time.Second)
			l.conn.WriteMessage(websocket.PingMessage, buf)
		}
	}()

	l.conn.WriteMessage(websocket.TextMessage, buf)

	for {
		logger.Infof("waiting msg from websocket")
		msgType, raw, err := l.conn.ReadMessage()
		if err == nil {
			logger.Infof("receiving msg with type %v from websocket", msgType)
			switch msgType {
			case websocket.TextMessage:
				l.HandleText(string(raw))
			case websocket.BinaryMessage:
				l.HandleBinary(raw)
			case websocket.PingMessage:
				l.HandlePing(raw)
			case websocket.PongMessage:
				l.HandlePong(raw)
			case websocket.CloseMessage:
				l.HandleClose(raw)
			default:
				logger.Errorf("unsupported message type: %v", msgType)
			}
			continue
		}

		logger.Errorf("websocket connection lost, error: %v", err.Error())
		// try to reconnect with backoff
		bo := backoff.NewExponentialBackOff()
		err = backoff.Retry(l.setupWebsocketConn, bo)
		if err != nil {
			logger.Errorf("failed to reconnect websocket, error: %v", err.Error())
			return
		}
		logger.Infof("websocket reconnect success")
		l.conn.WriteMessage(websocket.TextMessage, buf)
	}
}

func (l *LarkGuardBot) setupWebsocketConn() error {
	c, rsp, err := websocket.DefaultDialer.Dial(wsAdapterEndpoint, nil)
	if err != nil {
		logger.Fatalf("failed to connect ws, endpoint: %v, error: %v", wsAdapterEndpoint, rsp)
		return err
	}
	l.conn = c
	return nil
}

func (l *LarkGuardBot) HandlePing(data []byte) {
	logger.Debugf("receive 'pong' msg: %v", string(data))

	err := l.conn.WriteMessage(websocket.PongMessage, data)
	if err != nil {
		logger.Errorf("failed to write pong, error: %v", err.Error())
		return
	}
}

func (l *LarkGuardBot) HandlePong(data []byte) {
	logger.Infof("receive 'pong' msg: %v", string(data))
}

func (l *LarkGuardBot) HandleBinary(data []byte) {
	logger.Infof("receive binary msg: %v", string(data))
}

func (l *LarkGuardBot) HandleClose(data []byte) {
	logger.Infof("receive close msg: %v", string(data))
}

func (l *LarkGuardBot) HandleText(data string) {
	logger.Infof("receive text msg: %v", data)
	sj, err := simplejson.NewJson([]byte(data))
	if err != nil {
		logger.Errorf("unmarshal text msg failed, error: %v", err.Error())
		return
	}
	eventType := sj.Get("header").Get("event_type").MustString()
	if strings.Contains(eventType, "contact.user") {
		userMsg := new(lm.ContactUserMsg)
		if err := json.Unmarshal([]byte(data), userMsg); err != nil {
			logger.Errorf("failed to parse contact user message, error: %v", err.Error())
			return
		}
		lm.UserChan <- userMsg

		return
	}

	// Process lark department event
	if strings.Contains(eventType, "contact.department") {
		depMsg := new(lm.ContactDepMsg)
		if err = json.Unmarshal([]byte(data), depMsg); err != nil {
			logger.Errorf("failed to parse contact department message, error: %v", err.Error())
			return
		}
		lm.DepChan <- depMsg
		return
	}
}
