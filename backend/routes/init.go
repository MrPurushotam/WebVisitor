package routes

import (
	"github.com/gin-gonic/gin"
)

func Init(r *gin.Engine) {
	v1 := r.Group("/api/v1")

	InitUserRouter(v1)
	InitUriRouter(v1)
	InitLogsRouter(v1)
}
