package routes

import "github.com/gin-gonic/gin"


func getAllLogs(c *gin.Context){

}


func InitLogsRouter(rg *gin.RouterGroup){
	router:= rg.Group("/logs")

	router.GET("/",getAllLogs)
}