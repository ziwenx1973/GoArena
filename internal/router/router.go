package router

import (
	"github.com/gin-gonic/gin"
	"github.com/ziwenx1973/GoArena/internal/handler"
	"github.com/ziwenx1973/GoArena/internal/middleware"
	"net/http"
)

func New(a *handler.AuthHandler, u *handler.UserHandler, g *handler.GameHandler) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())
	_ = r.SetTrustedProxies(nil)
	// Do not log request URLs: WebSocket query strings contain JWTs.
	r.Use(func(c *gin.Context) { c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 4096); c.Next() })
	r.GET("/", func(c *gin.Context) { c.File("web/index.html") })
	r.GET("/healthz", func(c *gin.Context) { c.JSON(200, gin.H{"status": "ok"}) })
	r.POST("/api/register", a.Register)
	r.POST("/api/login", a.Login)
	api := r.Group("/api", middleware.JWT(a.Auth))
	api.GET("/profile", u.Profile)
	api.GET("/records", u.Records)
	r.GET("/ws", g.Connect)
	return r
}
