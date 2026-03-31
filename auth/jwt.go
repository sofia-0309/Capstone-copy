package auth

import (
	"log"
	"time"

	"os"

	"github.com/golang-jwt/jwt/v5"
	"github.com/joho/godotenv"
)

func init() {
	//loading .env file
	err := godotenv.Load()
	if err != nil {
		log.Println("Warning: .env file not found")
	}
}

var jwtSecret = []byte(os.Getenv("JWT_SECRET"))

type JWTClaims struct {
	UserId   string `json:"userId"`
	Nickname string `json:"nickname"`
	IsAdmin  bool   `json:"isAdmin"`
	jwt.RegisteredClaims
}

// GenerateJWT for the user given some parameters
func GenerateJWT(userId, nickname string, isAdmin bool) (string, error) {
	claims := JWTClaims{
		UserId:   userId,
		Nickname: nickname,
		IsAdmin:  isAdmin,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}
