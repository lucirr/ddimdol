package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type SessionHandler struct{}

func NewSessionHandler() *SessionHandler {
	return &SessionHandler{}
}

func (h *SessionHandler) CreateSession(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"data": nil, "message": "TODO: create remote session"})
}

func (h *SessionHandler) ListSessions(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"data": []any{}, "message": "TODO: list remote sessions"})
}

func (h *SessionHandler) ActivateSession(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"data": nil, "message": "TODO: activate session " + c.Param("id")})
}

func (h *SessionHandler) TerminateSession(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"data": nil, "message": "TODO: terminate session " + c.Param("id")})
}

func (h *SessionHandler) GetRecording(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"data": nil, "message": "TODO: get recording for session " + c.Param("id")})
}
