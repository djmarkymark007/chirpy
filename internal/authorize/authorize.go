package authorize

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/djmarkymark007/chirpy/internal/database"
	"github.com/golang-jwt/jwt/v5"
)

func CreateJwt(id int, expiresRequest int, secret string) (string, error) {
	expires := 60 * 60
	if expiresRequest < expires && expiresRequest != 0 {
		expires = expiresRequest
	}

	claim := jwt.RegisteredClaims{Issuer: "chirpy",
		IssuedAt:  jwt.NewNumericDate(time.Now().UTC()),
		ExpiresAt: jwt.NewNumericDate(time.Now().UTC().Add(time.Duration(expires * int(time.Second)))),
		Subject:   fmt.Sprint(id)}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claim)
	jwtToken, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", err
	}
	return jwtToken, nil
}

func CreateRefreshToken() (string, error) {
	var rndValue [32]byte
	_, err := rand.Read(rndValue[:])
	refreshToken := hex.EncodeToString(rndValue[:])
	return refreshToken, err
}

func ValidateRefreshToken(token string, db *database.Database) (bool, database.UserDatabase, error) {
	var currentUser = database.UserDatabase{}

	users, err := db.GetUsers()
	if err != nil {
		return false, currentUser, err
	}

	validToken := false

	for _, user := range users {
		if user.RefreshToken == token {
			if user.TokenExpiresAt.After(time.Now().UTC()) {
				validToken = true
				currentUser = user
			} else {
				//NOTE(Mark): should this be logged?
				log.Print("time miss match")
			}
			break
		}
	}
	return validToken, currentUser, nil
}

func GetClaimFromJwt(token string, secret string) (*jwt.Token, error) {
	claims, err := jwt.ParseWithClaims(token, &jwt.RegisteredClaims{}, func(token *jwt.Token) (interface{}, error) {
		if token.Method != jwt.SigningMethodHS256 {
			// Not sure if this should be fatal or not
			log.Fatalf("Token Method: %v want: %v", token.Method, jwt.SigningMethodHS256)
		}
		return []byte(secret), nil
	})
	return claims, err
}

func GetIdFromJwt(token string, secret string) (int, error) {
	claims, err := GetClaimFromJwt(token, secret)
	if err != nil {
		return 0, err
	}

	idString, err := claims.Claims.GetSubject()
	if err != nil {
		return 0, err
	}

	return strconv.Atoi(idString)
}
