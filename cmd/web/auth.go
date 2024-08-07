package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/golang-jwt/jwt/v5"
)

const (
	jwtCookie   = "jwt"
	userIDClaim = "userID"
)

// Makes a JSON web token associated with userID.
// Signed with the key stored in environment variable "KEY".
func makeUserIDToken(userID string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		userIDClaim: userID,
	})
	return token.SignedString([]byte(os.Getenv("KEY")))
}

// Parses the provided JSON web token and retrieves the associated userID.
// Checks signature with the key stored in environment variable "KEY".
// Errors if the MAC is incorrect or if the token does not have a userID field.
func parseUserIDToken(tokenString string) (string, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
		}

		return []byte(os.Getenv("KEY")), nil
	})
	if err != nil {
		return "", err
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return "", errors.New("Could not get claims")
	}
	userIDInterface := claims[userIDClaim]
	userID, ok := userIDInterface.(string)
	if !ok {
		return "", errors.New("Could not get userID from claims")
	}
	return userID, nil
}
