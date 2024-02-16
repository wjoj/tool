package stringx

import (
	"crypto/md5"
	"crypto/subtle"
	"encoding/hex"
	"fmt"

	"golang.org/x/crypto/scrypt"
	"golang.org/x/text/width"
)

func PasswordEncrypt(salt, password string) string {
	dk, _ := scrypt.Key([]byte(password), []byte(salt), 32768, 8, 1, 32)
	return fmt.Sprintf("%x", string(dk))
}

func PasswordValidate(hashPwd1, hashPwd2 string) bool {
	return subtle.ConstantTimeCompare([]byte(hashPwd1), []byte(hashPwd2)) == 1
}

var md5c = md5.New()

func MD5(con string) string {
	md5c.Write([]byte(con))
	cipherStr := md5c.Sum(nil)
	return hex.EncodeToString(cipherStr)
}

// 获取字符串宽度
func CharacterWidth(s string) (w int) {
	for _, r := range s {
		switch width.LookupRune(r).Kind() {
		case width.EastAsianFullwidth, width.EastAsianWide:
			w += 2
		case width.EastAsianHalfwidth, width.EastAsianNarrow,
			width.Neutral, width.EastAsianAmbiguous:
			w += 1
		}
	}
	return w
}
