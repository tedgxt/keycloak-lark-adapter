package api

import (
	"github.com/gin-gonic/gin"
)

func SetupRouter() *gin.Engine {
	r := gin.Default()

	r.GET("/healthz", Healthz)

	r.POST("/api/v1/lark/notifications", Notifications)

	return r
}
