package stringx

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/des"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
)

//密文

func zeroPadding(ciphertext []byte, blockSize int) []byte {
	padding := blockSize - len(ciphertext)%blockSize
	padtext := bytes.Repeat([]byte{0}, padding)
	return append(ciphertext, padtext...)
}

func zeroUnPadding(origData []byte) []byte {
	return bytes.TrimFunc(origData,
		func(r rune) bool {
			return r == rune(0)
		})
}

// DESEncrypt 加密
func DESEncrypt(text string, key string) (string, error) {
	src := []byte(text)
	block, err := des.NewCipher([]byte(key))
	if err != nil {
		return "", err
	}
	bs := block.BlockSize()
	src = zeroPadding(src, bs)
	if len(src)%bs != 0 {
		return "", errors.New("need a multiple of the blocksize")
	}
	out := make([]byte, len(src))
	dst := out
	for len(src) > 0 {
		block.Encrypt(dst, src[:bs])
		src = src[bs:]
		dst = dst[bs:]
	}
	return hex.EncodeToString(out), nil
}

// DESEncrypt 加密
func DESEncryptBody(body []byte, key string) (string, error) {
	src := body
	block, err := des.NewCipher([]byte(key))
	if err != nil {
		return "", err
	}
	bs := block.BlockSize()
	src = zeroPadding(src, bs)
	if len(src)%bs != 0 {
		return "", errors.New("need a multiple of the blocksize")
	}
	out := make([]byte, len(src))
	dst := out
	for len(src) > 0 {
		block.Encrypt(dst, src[:bs])
		src = src[bs:]
		dst = dst[bs:]
	}
	return hex.EncodeToString(out), nil
}

// DESDecrypt 解密
func DESDecrypt(decrypted string, key string) (string, error) {
	src, err := hex.DecodeString(decrypted)
	if err != nil {
		return "", err
	}
	block, err := des.NewCipher([]byte(key))
	if err != nil {
		return "", err
	}
	out := make([]byte, len(src))
	dst := out
	bs := block.BlockSize()
	if len(src)%bs != 0 {
		return "", errors.New("crypto/cipher: input not full blocks")
	}
	for len(src) > 0 {
		block.Decrypt(dst, src[:bs])
		src = src[bs:]
		dst = dst[bs:]
	}
	out = zeroUnPadding(out)
	return string(out), nil
}

// DESDecrypt 解密
func DESDecryptBody(decrypted string, key string) ([]byte, error) {
	src, err := hex.DecodeString(decrypted)
	if err != nil {
		return []byte{}, err
	}
	block, err := des.NewCipher([]byte(key))
	if err != nil {
		return []byte{}, err
	}
	out := make([]byte, len(src))
	dst := out
	bs := block.BlockSize()
	if len(src)%bs != 0 {
		return []byte{}, errors.New("crypto/cipher: input not full blocks")
	}
	for len(src) > 0 {
		block.Decrypt(dst, src[:bs])
		src = src[bs:]
		dst = dst[bs:]
	}
	out = zeroUnPadding(out)
	return out, nil
}

func pKCS7Padding(ciphertext []byte, blocksize int) []byte {
	padding := blocksize - len(ciphertext)%blocksize
	padtext := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(ciphertext, padtext...)
}

// 去码
func pKCS7UnPadding(origData []byte) []byte {
	length := len(origData)
	unpadding := int(origData[length-1])
	return origData[:(length - unpadding)]
}

// AESEncrypt CBC加密 key 16,24,32长度
func AESEncrypt(orig string, key string) (origs string, errs error) {
	defer func() {
		if err := recover(); err != nil {
			errs = fmt.Errorf("%v", err)
		}
	}()
	// 转成字节数组
	origData := []byte(orig)
	k := []byte(key)
	// 分组秘钥
	// NewCipher该函数限制了输入k的长度必须为16, 24或者32
	block, _ := aes.NewCipher(k)
	// 获取秘钥块的长度
	blockSize := block.BlockSize()
	// 补全码
	origData = pKCS7Padding(origData, blockSize)
	// 加密模式
	blockMode := cipher.NewCBCEncrypter(block, k[:blockSize])
	// 创建数组
	cryted := make([]byte, len(origData))
	// 加密
	blockMode.CryptBlocks(cryted, origData)
	return base64.StdEncoding.EncodeToString(cryted), nil
}

// AESDecrypt CBC解密
func AESDecrypt(cryted string, key string) (origs string, errs error) {
	defer func() {
		if err := recover(); err != nil {
			errs = fmt.Errorf("%v", err)
		}
	}()

	// 转成字节数组
	crytedByte, _ := base64.StdEncoding.DecodeString(cryted)
	k := []byte(key)
	// 分组秘钥
	block, _ := aes.NewCipher(k)
	// 获取秘钥块的长度
	blockSize := block.BlockSize()
	// 加密模式
	blockMode := cipher.NewCBCDecrypter(block, k[:blockSize])
	// 创建数组
	orig := make([]byte, len(crytedByte))
	// 解密
	blockMode.CryptBlocks(orig, crytedByte)
	// 去补全码
	orig = pKCS7UnPadding(orig)
	return string(orig), nil
}

func generateKey(key []byte) (genKey []byte) {
	genKey = make([]byte, 16)
	copy(genKey, key)
	for i := 16; i < len(key); {
		for j := 0; j < 16 && i < len(key); j, i = j+1, i+1 {
			genKey[j] ^= key[i]
		}
	}
	return genKey
}

// AESEncryptModelECB ECB加密
func AESEncryptModelECB(src []byte, key []byte) (encrypted []byte, errs error) {
	defer func() {
		if err := recover(); err != nil {
			errs = fmt.Errorf("%v", err)
		}
	}()
	cipher, _ := aes.NewCipher(generateKey(key))
	length := (len(src) + aes.BlockSize) / aes.BlockSize
	plain := make([]byte, length*aes.BlockSize)
	copy(plain, src)
	pad := byte(len(plain) - len(src))
	for i := len(src); i < len(plain); i++ {
		plain[i] = pad
	}
	encrypted = make([]byte, len(plain))
	// 分组分块加密
	for bs, be := 0, cipher.BlockSize(); bs <= len(src); bs, be = bs+cipher.BlockSize(), be+cipher.BlockSize() {
		cipher.Encrypt(encrypted[bs:be], plain[bs:be])
	}

	return encrypted, nil
}

// AESDecryptModelECB ECB解密
func AESDecryptModelECB(encrypted []byte, key []byte) (decrypted []byte, errs error) {
	defer func() {
		if err := recover(); err != nil {
			errs = fmt.Errorf("%v", err)
		}
	}()
	cipher, _ := aes.NewCipher(generateKey(key))
	decrypted = make([]byte, len(encrypted))
	//
	for bs, be := 0, cipher.BlockSize(); bs < len(encrypted); bs, be = bs+cipher.BlockSize(), be+cipher.BlockSize() {
		cipher.Decrypt(decrypted[bs:be], encrypted[bs:be])
	}

	trim := 0
	if len(decrypted) > 0 {
		trim = len(decrypted) - int(decrypted[len(decrypted)-1])
	}

	return decrypted[:trim], nil
}

// AESDecrypterModelCFB CFB加密
func AESDecrypterModelCFB(orig string, keys string) (origs string, errs error) {

	key, errs := hex.DecodeString(keys)
	if errs != nil {
		return
	}
	ciphertext, errs := hex.DecodeString(orig)
	if errs != nil {
		return
	}
	block, errs := aes.NewCipher(key)
	if errs != nil {
		return
	}

	if len(ciphertext) < aes.BlockSize {
		errs = fmt.Errorf("ciphertext too short")
		return
	}
	iv := ciphertext[:aes.BlockSize]
	ciphertext = ciphertext[aes.BlockSize:]
	defer func() {
		if err := recover(); err != nil {
			errs = fmt.Errorf("%v", err)
		}
	}()
	stream := cipher.NewCFBDecrypter(block, iv)

	stream.XORKeyStream(ciphertext, ciphertext)
	return string(ciphertext), nil
}

// AESEncryptModelCFB CFB解密
func AESEncryptModelCFB(orig string, keys string) (origs string, errs error) {
	key, _ := hex.DecodeString(keys)
	plaintext := []byte(orig)

	block, errs := aes.NewCipher(key)
	if errs != nil {
		return
	}

	ciphertext := make([]byte, aes.BlockSize+len(plaintext))
	iv := ciphertext[:aes.BlockSize]
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		errs = err
		return
	}
	defer func() {
		if err := recover(); err != nil {
			errs = fmt.Errorf("%v", err)
		}
	}()
	stream := cipher.NewCFBEncrypter(block, iv)
	stream.XORKeyStream(ciphertext[aes.BlockSize:], plaintext)
	return string(ciphertext), nil
}

// RSAEncrypt rsa加密 公钥(根据私钥生成:openssl rsa -in rsa_private_key.pem -pubout -out rsa_public_key.pem)
func RSAEncrypt(origData []byte, publicKey []byte) ([]byte, error) {
	block, _ := pem.Decode(publicKey)
	if block == nil {
		return nil, errors.New("public key error")
	}
	// 解析公钥
	pubInterface, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	// 类型断言
	pub := pubInterface.(*rsa.PublicKey)
	//加密
	return rsa.EncryptPKCS1v15(rand.Reader, pub, origData)
}

// RSADecrypt rsa解密  私钥(私钥生成:openssl genrsa -out rsa_private_key.pem 1024)
func RSADecrypt(ciphertext []byte, privateKey []byte) ([]byte, error) {
	block, _ := pem.Decode(privateKey)
	if block == nil {
		return nil, fmt.Errorf("%v", "private key error!")
	}
	//解析PKCS1格式的私钥
	priv, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	// 解密
	return rsa.DecryptPKCS1v15(rand.Reader, priv, ciphertext)
}

/*
RSAEncryptSuper  超过key长度加密
kLng: key长度
*/
func RSAEncryptSuper(origData []byte, publicKey []byte, kLng int) ([]byte, error) {
	baseLng := kLng/8 - 11
	dLng := len(origData)
	Encs := []byte{}
	offset := 0
	size := baseLng
	for {
		end := offset + size
		if end >= dLng {
			end = dLng
		}
		if offset >= dLng {
			offset = dLng
		}
		Enc, err := RSAEncrypt(origData[offset:end], publicKey)
		if err != nil {
			return []byte{}, fmt.Errorf("RSA加密错误:%v", err)
		}
		Encs = append(Encs, Enc...)
		if dLng == end {
			break
		}
		offset += size
	}
	return Encs, nil
}

/*
RSADecryptSuper 超过key长度解密
kLng: key长度
*/
func RSADecryptSuper(ciphertext []byte, privateKey []byte, kLng int) ([]byte, error) {
	baseLng := kLng / 8
	dLng := len(ciphertext)
	offset := 0
	size := baseLng
	body := []byte{}
	for {
		end := offset + size
		if end >= dLng {
			end = dLng
		}
		if offset >= dLng {
			offset = dLng
		}
		Enc, err := RSADecrypt(ciphertext[offset:end], privateKey)
		if err != nil {
			return []byte{}, fmt.Errorf("RSA解密错误:%v", err)
		}
		body = append(body, Enc...)
		if dLng == end {
			break
		}
		offset += size
	}
	return body, nil
}
