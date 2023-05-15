package conversion

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net"
	"reflect"
	"strconv"
	"strings"
	"unsafe"
)

func ToString(value interface{}) string {
	v := reflect.TypeOf(value).Kind()
	switch v { //v := value.(type)
	case reflect.String:
		return value.(string)
	case reflect.Int:
		return strconv.FormatInt(int64(v), 10)
	case reflect.Int8:
		return strconv.FormatInt(int64(v), 10)
	case reflect.Int16:
		return strconv.FormatInt(int64(v), 10)
	case reflect.Int32:
		return strconv.FormatInt(int64(v), 10)
	case reflect.Int64:
		return strconv.FormatInt(int64(v), 10)
	case reflect.Uint:
		return strconv.FormatUint(uint64(v), 10)
	case reflect.Uint8:
		return strconv.FormatUint(uint64(v), 10)
	case reflect.Uint16:
		return strconv.FormatUint(uint64(v), 10)
	case reflect.Uint32:
		return strconv.FormatUint(uint64(v), 10)
	case reflect.Uint64:
		return strconv.FormatUint(uint64(v), 10)
	case reflect.Float64:
		return strconv.FormatFloat(float64(v), 'f', 0, 64)
	case reflect.Float32:
		return strconv.FormatFloat(float64(v), 'f', 0, 32)
	case reflect.Array, reflect.Slice:
		switch value := value.(type) {
		case []byte:
			return ToStringByBytes(value)
		}
		fallthrough
	case reflect.Map, reflect.Struct:
		bt, errors := json.Marshal(value)
		if errors != nil {
			return errors.Error()
		} else {
			return ToStringByBytes(bt)
		}

	default:
		return fmt.Sprintf("%v", value)
	}
}

func ToInt(value interface{}) int {
	var number int
	switch v := value.(type) {
	case int:
		number = int(v)
	case int64:
		number = int(int64(v))
	case float64:
		number = int(float64(v))
	case float32:
		number = int(float32(v))
	case string:
		number, _ = strconv.Atoi(string(v))
	default:
		number = 0
	}
	return number
}

func ToInt64(num interface{}) int64 {
	var number int64
	switch num := num.(type) {
	case int:
		number = int64(num)
	case float64:
		number = int64(num)
	case float32:
		number = int64(num)
	case string:
		number, _ = strconv.ParseInt(num, 10, 64)
	case interface{}:
		number, _ = strconv.ParseInt(fmt.Sprintf("%v", num), 10, 64)
	}
	return number
}

func ToFloat64(value string) float64 {
	f, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0.0
	}
	return f
}

func ToObjFromByte(body []byte, v interface{}) error {
	errs := json.Unmarshal(body, &v)
	if errs != nil {
		return errs
	}

	return nil
}

func ToObjFrom(value string, v interface{}) error {
	return ToObjFromByte([]byte(value), v)
}

func ToBytes(value interface{}) ([]byte, error) {
	return json.Marshal(value)
}

func ToIPv4(ip int64) string {
	return net.IP{byte(ip >> 24), byte(ip >> 16), byte(ip >> 8), byte(ip)}.String()
}

func ToIPv4AtoN(ip string) int64 {
	ret := big.NewInt(0)
	ret.SetBytes(net.ParseIP(ip).To4())
	return ret.Int64()
}

func ToIPv6(numasstr string) (string, error) {
	bi, ok := new(big.Int).SetString(numasstr, 10)
	if !ok {
		return "", errors.New("fail to convert string to big.Int")
	}
	b255 := new(big.Int).SetBytes([]byte{255})
	var buf = make([]byte, 2)
	p := make([]string, 8)
	j := 0
	var i uint
	tmpint := new(big.Int)
	for i = 0; i < 16; i += 2 {
		tmpint.Rsh(bi, 120-i*8).And(tmpint, b255)
		bytes := tmpint.Bytes()
		if len(bytes) > 0 {
			buf[0] = bytes[0]
		} else {
			buf[0] = 0
		}
		tmpint.Rsh(bi, 120-(i+1)*8).And(tmpint, b255)
		bytes = tmpint.Bytes()
		if len(bytes) > 0 {
			buf[1] = bytes[0]
		} else {
			buf[1] = 0
		}
		p[j] = hex.EncodeToString(buf)
		j++
	}
	return strings.Join(p, ":"), nil
}

func ToIPv6AtoN(ip string) string {
	ret := big.NewInt(0)
	ret.SetBytes(net.ParseIP(ip).To16())
	return ret.String()
}

func ToBase64Encode(input []byte) string {
	return base64.StdEncoding.EncodeToString(input)
}

func ToBase64Decode(base64s string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(base64s)
}

func ToBase64URLEncoding(str string) string {
	return base64.URLEncoding.EncodeToString([]byte(str))
}

func ToBase64URLDecode(str string) ([]byte, error) {
	return base64.URLEncoding.DecodeString(str)
}

func ToJoint(sep string, s ...string) string {
	return strings.Join(s, sep)
}

func ToJointFunc(sep string, lng int, sF func(index int) string) string {
	var str strings.Builder

	for i := 0; i < lng; i++ {
		s := sF(i)
		if len(s) == 0 {
			continue
		}
		if str.Len() == 0 {
			str.WriteString(sF(i))
		} else {
			str.WriteString(sep)
			str.WriteString(sF(i))
		}
	}
	return str.String()
}

func ToBytesNs(v ...any) ([]byte, error) {
	buff := new(bytes.Buffer)
	for _, vc := range v {
		err := binary.Write(buff, binary.BigEndian, vc)
		if err != nil {
			return nil, err
		}
	}
	return buff.Bytes(), nil
}

func ToNsBytes(b []byte, v ...any) error {
	buff := bytes.NewBuffer(b)
	for _, vc := range v {
		err := binary.Read(buff, binary.BigEndian, vc)
		if err != nil {
			return err
		}
	}
	return nil
}

func ToBytesByString(val string) []byte {
	return *(*[]byte)(unsafe.Pointer(&val))
}

func ToStringByBytes(b []byte) string {
	return *(*string)(unsafe.Pointer(&b))
}
