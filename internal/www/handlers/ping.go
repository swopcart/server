package handlers

import "github.com/gin-gonic/gin"

func Root(c *gin.Context) {
	c.String(200, "Hello world!")
}

func Ping(c *gin.Context) {
	c.JSON(200, gin.H{
		"message": "pong",
	})
}
