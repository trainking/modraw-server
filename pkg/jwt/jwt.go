package jwt

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt"
)

type Claims struct {
	UserID   string `json:"user_id"`
	Email    string `json:"email"`
	Nickname string `json:"nickname"`
	jwt.StandardClaims
}

type RefreshClaims struct {
	UserID  string `json:"user_id"`
	TokenID string `json:"token_id"`
	jwt.StandardClaims
}

type ShareClaims struct {
	CanvasID   string `json:"canvas_id"`
	Permission string `json:"permission"`
	jwt.StandardClaims
}

func GenerateAccessToken(userID, email, nickname, secret string, ttl time.Duration) (string, error) {
	now := time.Now()
	claims := Claims{
		UserID:   userID,
		Email:    email,
		Nickname: nickname,
		StandardClaims: jwt.StandardClaims{
			IssuedAt:  now.Unix(),
			ExpiresAt: now.Add(ttl).Unix(),
			Subject:   userID,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

func GenerateRefreshToken(userID, secret string, ttl time.Duration) (token string, tokenID string, err error) {
	now := time.Now()
	tokenID = fmt.Sprintf("%d", now.UnixNano())
	claims := RefreshClaims{
		UserID:  userID,
		TokenID: tokenID,
		StandardClaims: jwt.StandardClaims{
			IssuedAt:  now.Unix(),
			ExpiresAt: now.Add(ttl).Unix(),
			Subject:   userID,
			Id:        tokenID,
		},
	}

	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	token, err = t.SignedString([]byte(secret))
	return
}

func ValidateToken(tokenStr string, secret string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(secret), nil
	})
	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token claims")
	}

	return claims, nil
}

func GenerateShareToken(canvasID, permission, secret string, ttl time.Duration) (string, error) {
	now := time.Now()
	claims := ShareClaims{
		CanvasID:   canvasID,
		Permission: permission,
		StandardClaims: jwt.StandardClaims{
			IssuedAt:  now.Unix(),
			ExpiresAt: now.Add(ttl).Unix(),
			Subject:   canvasID,
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

func ValidateShareToken(tokenStr string, secret string) (*ShareClaims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &ShareClaims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(secret), nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*ShareClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid share token claims")
	}
	return claims, nil
}

func ValidateRefreshToken(tokenStr string, secret string) (*RefreshClaims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &RefreshClaims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(secret), nil
	})
	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*RefreshClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid refresh token claims")
	}

	return claims, nil
}
