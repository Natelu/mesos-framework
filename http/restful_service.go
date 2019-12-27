package http

import (
	"mine-mesos-framework/framework"
	"net/http"

	"github.com/gin-gonic/gin"
)

var (
	cluster *framework.Cluster
)

func addTask(c *gin.Context) {
	var taskInfo *framework.TaskInfo
	if err := c.ShouldBindJSON(taskInfo); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	cluster.SetTask(taskInfo)
	c.String(http.StatusOK, "")
}

func setTask(c *gin.Context) {
	c.String(http.StatusOK, "message")
}
