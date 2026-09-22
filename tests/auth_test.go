package tests

import (
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/ziwenx1973/GoArena/internal/middleware"
	"github.com/ziwenx1973/GoArena/internal/service"
	"net/http/httptest"
	"testing"
	"time"
)

func TestJWTMiddleware(t *testing.T) {
	auth := &service.AuthService{Secret: "unit-test-only-secret-not-for-running-server"}
	claims := service.Claims{UserID: 7, Username: "alice", RegisteredClaims: jwt.RegisteredClaims{Issuer: "go-arena", ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour))}}
	valid, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(auth.Secret))
	if err != nil {
		t.Fatal(err)
	}
	claims.ExpiresAt = jwt.NewNumericDate(time.Now().Add(-time.Hour))
	expired, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(auth.Secret))
	claims.ExpiresAt = nil
	noExpiry, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(auth.Secret))
	claims.ExpiresAt = jwt.NewNumericDate(time.Now().Add(time.Hour))
	wrongAlgorithm, _ := jwt.NewWithClaims(jwt.SigningMethodHS384, claims).SignedString([]byte(auth.Secret))
	r := gin.New()
	r.GET("/", middleware.JWT(auth), func(c *gin.Context) {
		if c.GetUint("user_id") != 7 || c.GetString("username") != "alice" {
			t.Error("claims not in context")
		}
		c.Status(204)
	})
	for _, tc := range []struct {
		name, header string
		want         int
	}{{"valid", "Bearer " + valid, 204}, {"missing", "", 401}, {"bad", "Bearer invalid", 401}, {"expired", "Bearer " + expired, 401}, {"missing expiration", "Bearer " + noExpiry, 401}, {"algorithm", "Bearer " + wrongAlgorithm, 401}, {"scheme", "Basic " + valid, 401}} {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", nil)
			req.Header.Set("Authorization", tc.header)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			if w.Code != tc.want {
				t.Fatalf("status %d want %d", w.Code, tc.want)
			}
		})
	}
}
