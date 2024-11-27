package utils

import (
	"errors"
	"strings"

	"github.com/golang-jwt/jwt"
)

func ExtractUserIDFromToken(authHeader string) (uint, error) {
    parts := strings.SplitN(authHeader, " ", 2)
    if len(parts) != 2 || parts[0] != "Bearer" {
        return 0, errors.New("invalid authorization header format")
    }

    tokenString := parts[1]

    token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
        return JwtSecret, nil
    })

    if err != nil || !token.Valid {
        return 0, errors.New("invalid token")
    }

    claims, ok := token.Claims.(jwt.MapClaims)
    if !ok {
        return 0, errors.New("invalid token claims")
    }

    userIDFloat, ok := claims["user_id"].(float64)
    if !ok {
        return 0, errors.New("invalid user ID in token")
    }

    return uint(userIDFloat), nil
}
