package handler

import (
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/ziwenx1973/GoArena/internal/service"
	"gorm.io/gorm"
	"log/slog"
)

type AuthHandler struct{ Auth *service.AuthService }
type credentials struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (h *AuthHandler) Register(c *gin.Context) {
	var in credentials
	if c.ShouldBindJSON(&in) != nil {
		c.JSON(400, gin.H{"error": "invalid JSON"})
		return
	}
	u, err := h.Auth.Register(c.Request.Context(), in.Username, in.Password)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(201, u)
}
func (h *AuthHandler) Login(c *gin.Context) {
	var in credentials
	if c.ShouldBindJSON(&in) != nil {
		c.JSON(400, gin.H{"error": "invalid JSON"})
		return
	}
	token, err := h.Auth.Login(c.Request.Context(), in.Username, in.Password)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(200, gin.H{"token": token})
}
func respondError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrInput):
		c.JSON(400, gin.H{"error": err.Error()})
	case errors.Is(err, service.ErrCredentials):
		c.JSON(401, gin.H{"error": err.Error()})
	case errors.Is(err, gorm.ErrDuplicatedKey):
		c.JSON(409, gin.H{"error": "username already exists"})
	case errors.Is(err, gorm.ErrRecordNotFound):
		c.JSON(404, gin.H{"error": "user not found"})
	default:
		slog.Error("Database error", "error", err)
		c.JSON(500, gin.H{"error": "database operation failed"})
	}
}
