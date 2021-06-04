package config

import (
	"log"
	"os"
)

var (
	LogLevel string

	// Keycloak related config
	Host         string
	ClientId     string
	ClientSecret string
	Realm        string

	// Lark related config
	AppId             string
	AppSecret         string
	VerificationToken string
	EncryptKey        string

	// EventSource support "http" and "websocket"
	EventSource string
	// ServerPort default 8080
	ServerPort string
)

func Init() {
	LogLevel = os.Getenv("LOG_LEVEL")

	Host = os.Getenv("KEYCLOAK_HOST")
	if len(Host) == 0 {
		log.Fatalf("cannot get param KEYCLOAK_HOST from env")
	}

	ClientId = os.Getenv("KEYCLOAK_CLIENT_ID")
	if len(ClientId) == 0 {
		log.Fatalf("cannot get param KEYCLOAK_CLIENT_ID from env")
	}

	ClientSecret = os.Getenv("KEYCLOAK_CLIENT_SECRET")
	if len(ClientSecret) == 0 {
		log.Fatalf("cannot get param KEYCLOAK_CLIENT_SECRET from env")
	}

	Realm = os.Getenv("KEYCLOAK_REALM")
	if len(Realm) == 0 {
		log.Fatalf("cannot get param KEYCLOAK_REALM from env")
	}

	AppId = os.Getenv("LARK_APP_ID")
	if len(AppId) == 0 {
		log.Fatalf("cannot get param LARK_APP_ID from env")
	}

	AppSecret = os.Getenv("LARK_APP_SECRET")
	if len(AppSecret) == 0 {
		log.Fatalf("cannot get param LARK_APP_SECRET from env")
	}

	VerificationToken = os.Getenv("LARK_VERIFICATION_TOKEN")
	if len(VerificationToken) == 0 {
		log.Fatalf("cannot get param LARK_VERIFICATION_TOKEN from env")
	}

	EncryptKey = os.Getenv("LARK_ENCRYPT_KEY")

	EventSource = os.Getenv("EVENT_RESOURCE")
	if len(EventSource) == 0 {
		EventSource = "websocket"
	}

	ServerPort = os.Getenv("SERVER_PORT")
	if len(ServerPort) == 0 {
		ServerPort = "8080"
	}

}
