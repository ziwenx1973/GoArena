package service

import (
	"context"
	"errors"
	"github.com/golang-jwt/jwt/v5"
	"github.com/ziwenx1973/GoArena/internal/model"
	"github.com/ziwenx1973/GoArena/internal/repository"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"regexp"
	"time"
)

var ErrCredentials = errors.New("invalid username or password")
var ErrInput = errors.New("username must be 3-32 ASCII letters, digits or underscores; password must be 6-72 bytes")
var usernamePattern = regexp.MustCompile(`^[a-zA-Z0-9_]{3,32}$`)

type Claims struct {
	UserID   uint   `json:"user_id"`
	Username string `json:"username"`
	jwt.RegisteredClaims
}
type AuthService struct {
	Users  *repository.UserRepository
	Secret string
}

func (s *AuthService) Register(ctx context.Context, name, password string) (*model.User, error) {
	if !usernamePattern.MatchString(name) || len(password) < 6 || len(password) > 72 {
		return nil, ErrInput
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	u := &model.User{Username: name, PasswordHash: string(hash)}
	return u, s.Users.Create(ctx, u)
}
func (s *AuthService) Login(ctx context.Context, name, password string) (string, error) {
	if !usernamePattern.MatchString(name) || len(password) < 6 || len(password) > 72 {
		return "", ErrCredentials
	}
	u, err := s.Users.ByName(ctx, name)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return "", ErrCredentials
	}
	if err != nil {
		return "", err
	}
	if bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)) != nil {
		return "", ErrCredentials
	}
	now := time.Now()
	claims := Claims{u.ID, u.Username, jwt.RegisteredClaims{ExpiresAt: jwt.NewNumericDate(now.Add(24 * time.Hour)), IssuedAt: jwt.NewNumericDate(now), Issuer: "go-arena"}}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(s.Secret))
}
func (s *AuthService) Parse(token string) (*Claims, error) {
	c := &Claims{}
	t, err := jwt.ParseWithClaims(token, c, func(*jwt.Token) (any, error) { return []byte(s.Secret), nil }, jwt.WithValidMethods([]string{"HS256"}), jwt.WithExpirationRequired(), jwt.WithIssuer("go-arena"))
	if err != nil || !t.Valid || c.UserID == 0 || c.Username == "" {
		return nil, ErrCredentials
	}
	return c, nil
}
