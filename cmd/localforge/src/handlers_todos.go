package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func (s *Server) handleGetTodos(c *gin.Context) {
	todos := s.todoMgr.GetTodos()
	c.JSON(http.StatusOK, gin.H{
		"todos": todos,
	})
}
