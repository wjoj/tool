package stringx

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"

	"golang.org/x/crypto/scrypt"
)

func PasswordEncrypt(salt, password string) string {
	dk, _ := scrypt.Key([]byte(password), []byte(salt), 32768, 8, 1, 32)
	return fmt.Sprintf("%x", string(dk))
}

var md5c = md5.New()

func MD5(con string) string {
	md5c.Write([]byte(con))
	cipherStr := md5c.Sum(nil)
	return hex.EncodeToString(cipherStr)
}
