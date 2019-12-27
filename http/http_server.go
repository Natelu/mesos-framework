package http

import (
	"mine-mesos-framework/framework"

	"github.com/gin-gonic/gin"
	//h "net/http"
)

type HttpServer struct {
	route *gin.Engine
}

func NewHttpServer(fc *framework.Cluster) (server *HttpServer) {
	cluster = fc
	server = &HttpServer{
		route: gin.Default(),
	}
	server.initRoute()
	return
}

func (server *HttpServer) initRoute() {
	server.route.GET("/ver", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"version": "0.1",
		})
	})

	// server.route.GET("/tasks", getTasks)
	// server.route.GET("/tasks/:name", getTask)
	server.route.POST("/tasks", addTask)
	server.route.PUT("/tasks/:name", setTask)
	// server.route.DELETE("/tasks/:name", removeTask)

	// server.route.GET("/overtasks", getOverTasks)

	// server.route.POST("/loginJSON", func(c *gin.Context) {
	// 	var json Login
	// 	if err := c.ShouldBindJSON(&json); err != nil {
	// 		c.JSON(h.StatusBadRequest, gin.H{"error": err.Error()})
	// 		return
	// 	}

	// 	if json.User != "manu" || json.Password != "123" {
	// 		c.JSON(h.StatusUnauthorized, gin.H{"status": "unauthorized"})
	// 		return
	// 	}

	// 	c.JSON(h.StatusOK, gin.H{"status": "you are logged in"})
	// })
}

func (server *HttpServer) Run(addr *string) {
	server.route.Run(*addr)
}
