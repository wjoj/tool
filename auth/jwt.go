package auth

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
)

const Authorization = "Authorization"

type Jwt struct {
	Secret string `json:"secret" yaml:"secret"`
	Expire int64  `json:"expire" yaml:"expire"`
}

func (j *Jwt) GenerateToken(m map[string]any) (string, int64, error) {
	now := time.Now().Unix()
	exp := now + j.Expire
	claims := make(jwt.MapClaims)
	for k, v := range m {
		claims[k] = v
	}
	claims["exp"] = exp
	claims["iat"] = now
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tn, err := token.SignedString([]byte(j.Secret))
	if err != nil {
		return "", 0, err
	}
	return tn, exp, nil
}

func (j *Jwt) ParseToken(token string) (jwt.MapClaims, error) {
	tokenClaims, err := jwt.ParseWithClaims(token, &jwt.MapClaims{}, func(token *jwt.Token) (interface{}, error) {
		return j.Secret, nil
	})
	if err != nil {
		return nil, err
	}
	if tokenClaims == nil {
		return nil, errors.New("token claims is empty")
	}
	if !tokenClaims.Valid {
		return nil, errors.New("token valid fail")
	}
	claims, ok := tokenClaims.Claims.(jwt.MapClaims)
	if ok {
		return claims, nil
	}
	return nil, errors.New("token fail")
}

type JwtUser struct {
	Jwt
}

func JwtGinParseToken(j *JwtUser) func(g *gin.Context) {
	return func(g *gin.Context) {
		token := g.GetHeader(Authorization)
		if len(token) == 0 {
			g.AbortWithError(int(http.StatusUnauthorized), errors.New("authorization can't be empty"))
			return
		}
		clm, err := j.ParseToken(token)
		if err != nil {
			g.AbortWithError(int(http.StatusUnauthorized), err)
			return
		}
		key := "userId"
		userId, is := clm[key]
		if !is {
			g.AbortWithError(int(http.StatusUnauthorized), errors.New("userId field does not exist"))
			return
		}
		g.Request = g.Request.WithContext(context.WithValue(g.Request.Context(), &key, userId))
		g.Next()
	}
}
