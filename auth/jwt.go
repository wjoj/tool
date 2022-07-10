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
		return []byte(j.Secret), nil
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
	switch v := tokenClaims.Claims.(type) {
	case jwt.MapClaims:
		return jwt.MapClaims(v), nil
	case *jwt.MapClaims:
		return *(*jwt.MapClaims)(v), nil
	}
	return nil, errors.New("token fail")
}

var jwtMapClaims = "jwtMapClaims"

func JwtGinParseToken(j *Jwt) func(g *gin.Context) {
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
		g.Request = g.Request.WithContext(context.WithValue(g.Request.Context(), &jwtMapClaims, clm))
		g.Next()
	}
}

func JwtGinMapClaims(g *gin.Context) map[string]any {
	cla := g.Request.Context().Value(&jwtMapClaims)
	switch v := cla.(type) {
	case jwt.MapClaims:
		return jwt.MapClaims(v)
	case *jwt.MapClaims:
		return *(*jwt.MapClaims)(v)
	case map[string]any:
		return map[string]any(v)
	}
	return nil
}
