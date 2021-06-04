## Keycloak Lark Adapter

This repository is a work in progress and contains the source code for synchronizing data from lark to keycloak.

## Supported Lark Events

- User create
- User enable/disable
- User delete
- User update
- Department create
- Department update
- Department delete

## Start Application

1. Set the configurations in env.

   - LARK_APP_ID
   - LARK_APP_SECRET
   - LARK_VERIFICATION_TOKEN
   - KEYCLOAK_HOST
   - KEYCLOAK_CLIENT_ID
   - KEYCLOAK_CLIENT_SECRET
   - KEYCLOAK_REALM
   - WEBSOCKET_ADAPTER_ENDPOINT
   - LOG_LEVEL
   - EVENT_RESOURCE
   - SERVER_PORT
   
2. Start `main()` function in `cmd/cmd.go`