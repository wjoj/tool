package jwt

import (
	"errors"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/wjoj/tool/v2/log"
	"github.com/wjoj/tool/v2/utils"
)

const (
	Contextlaims = "claims"
)

type Config struct {
	Method  string        `json:"method" yaml:"method"`
	Secret  string        `json:"secret" yaml:"secret"`   //密钥
	Public  string        `json:"public" yaml:"public"`   //公钥
	Private string        `json:"private" yaml:"private"` //私钥
	Expire  time.Duration `json:"expire" yaml:"expire"`   //过期时间
	Number  int64         `json:"number" yaml:"number"`   //token数量 超过删除前面token 默认0:不限制并可控制token -1:不限制
}

type JwtToken struct {
	Token  string `json:"token"`
	Expire int64  `json:"expire"` //过期时间
}

type Claims[T any] struct {
	jwt.RegisteredClaims
	Data T `json:"data"`
}

type Jwt struct {
	cfg *Config
	pub any
	pri any
}

func New(cfg *Config) (*Jwt, error) {
	if len(cfg.Method) == 0 {
		cfg.Method = jwt.SigningMethodHS256.Alg()
	} else {
		cfg.Method = strings.ToUpper(string(cfg.Method))
	}
	if strings.HasPrefix(string(cfg.Method), "ED") {
		cfg.Method = jwt.SigningMethodEdDSA.Alg()
	}
	jt := &Jwt{
		cfg: cfg,
	}
	if strings.HasPrefix(string(cfg.Method), "RS") {
		pubBy, priBy, err := readPubPri(cfg.Public, cfg.Private)
		if err != nil {
			return nil, err
		}
		jt.pub, err = jwt.ParseRSAPublicKeyFromPEM(pubBy)
		if err != nil {
			return nil, err
		}
		jt.pri, err = jwt.ParseRSAPrivateKeyFromPEM(priBy)
		if err != nil {
			return nil, err
		}
	} else if strings.HasPrefix(string(cfg.Method), "Ed") {
		pubBy, priBy, err := readPubPri(cfg.Public, cfg.Private)
		if err != nil {
			return nil, err
		}
		jt.pub, err = jwt.ParseEdPublicKeyFromPEM(pubBy)
		if err != nil {
			return nil, err
		}
		jt.pri, err = jwt.ParseEdPrivateKeyFromPEM(priBy)
		if err != nil {
			return nil, err
		}
	} else if strings.HasPrefix(string(cfg.Method), "ES") {
		pubBy, priBy, err := readPubPri(cfg.Public, cfg.Private)
		if err != nil {
			return nil, err
		}
		jt.pub, err = jwt.ParseECPublicKeyFromPEM(pubBy)
		if err != nil {
			return nil, err
		}
		jt.pri, err = jwt.ParseECPrivateKeyFromPEM(priBy)
		if err != nil {
			return nil, err
		}
	} else {
		if len(cfg.Secret) == 0 {
			return nil, errors.New("secret is empty")
		}
		jt.pub = []byte(cfg.Secret)
		jt.pri = []byte(cfg.Secret)
	}
	return jt, nil
}

func readPubPri(p, pi string) (pub, pri []byte, err error) {
	pub, err = utils.FileRead(p)
	if err != nil {
		return
	}
	pri, err = utils.FileRead(pi)
	if err != nil {
		return
	}
	return
}

var cb *Jwt
var cbMap map[string]*Jwt
var defaultKey = utils.DefaultKey.DefaultKey

func Init(cfgs map[string]Config, options ...Option) error {
	log.Info("init jwt")
	opt := applyGenGormOptions(options...)
	defaultKey = opt.defKey.DefaultKey
	var err error
	cbMap, err = utils.Init("jwt", defaultKey, opt.defKey.Keys, cfgs, func(cfg Config) (*Jwt, error) {
		return New(&cfg)
	}, func(c *Jwt) {
		cb = c
	})
	if err != nil {
		log.Errorf("%v", err)
		return err
	}
	log.Info("init jwt success")
	return nil
}

func InitGlobal(cfg *Config) error {
	var err error
	cb, err = New(cfg)
	if err != nil {
		return err
	}
	return nil
}

func Get(key ...string) *Jwt {
	jt, err := utils.Get("jwt", defaultKey, func(s string) (*Jwt, bool) {
		cli, is := cbMap[s]
		return cli, is
	}, key...)
	if err != nil {
		panic(err)
	}
	return jt
}

func AuthMiddleware[T any](key ...string) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		token := ctx.GetHeader("Authorization")
		if len(token) == 0 {
			ctx.AbortWithStatusJSON(401, gin.H{"error": "Authorization header required"})
			return
		}
		token = strings.TrimPrefix(token, "Bearer ")
		claims, err := ValidateToken[T](token, key...)
		if err != nil {
			ctx.AbortWithStatusJSON(401, gin.H{"error": err.Error()})
			return
		}
		ctx.Set(Contextlaims, claims)
		ctx.Next()
	}
}

func GenerateToken[T any](uid string, data T, key ...string) (token *JwtToken, err error) {
	j := Get(key...)
	cl := Claims[T]{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(j.cfg.Expire)),
		},
		Data: data,
	}
	tk := jwt.NewWithClaims(jwt.GetSigningMethod(string(j.cfg.Method)), cl)
	s, err := tk.SignedString(j.pri)
	if err != nil {
		return nil, err
	}
	return &JwtToken{
		Token:  s,
		Expire: cl.ExpiresAt.Unix(),
	}, nil
}

func ValidateToken[T any](token string, key ...string) (claims *Claims[T], err error) {
	j := Get(key...)
	tokenc, err := jwt.ParseWithClaims(token, &Claims[T]{}, func(token *jwt.Token) (interface{}, error) {
		return j.pub, nil
	})
	if err != nil {
		return nil, err
	}
	return tokenc.Claims.(*Claims[T]), nil
}
