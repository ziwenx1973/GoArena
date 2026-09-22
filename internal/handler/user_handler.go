package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/ziwenx1973/GoArena/internal/service"
)

type UserHandler struct{ Users *service.UserService }

func (h *UserHandler) Profile(c *gin.Context) {
	u, err := h.Users.Profile(c.Request.Context(), c.GetUint("user_id"))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(200, u)
}
func (h *UserHandler) Records(c *gin.Context) {
	r, err := h.Users.Records(c.Request.Context(), c.GetUint("user_id"))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(200, gin.H{"records": r})
}
