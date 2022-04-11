package unique

import (
	"crypto/md5"
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"sync/atomic"
	"time"
)

//ObjectID id
type GUIDType string

//objectIDCounter 计数
var GUIDCounter = uint32(0)

//machineID id
var machineID = rmachineID()

func rmachineID() []byte {
	var sum [3]byte
	id := sum[:]
	hostname, err1 := os.Hostname()
	if err1 != nil {
		_, err2 := io.ReadFull(rand.Reader, id)
		if err2 != nil {
			panic(fmt.Errorf("cannot get hostname: %v; %v", err1, err2))
		}
		return id
	}
	hw := md5.New()
	hw.Write([]byte(hostname))
	copy(id, hw.Sum(nil))
	// fmt.Println("readMachineId:" + string(id))
	return id
}

/*
NewGUID 创建id
*/
func NewGUID() GUIDType {
	var b [12]byte
	// Timestamp, 4 bytes, big endian
	binary.BigEndian.PutUint32(b[:], uint32(time.Now().Unix()))
	// Machine, first 3 bytes of md5(hostname)
	b[4] = machineID[0]
	b[5] = machineID[1]
	b[6] = machineID[2]
	// Pid, 2 bytes, specs don't specify endianness, but we use big endian.
	pid := os.Getpid()
	b[7] = byte(pid >> 8)
	b[8] = byte(pid)
	// Increment, 3 bytes, big endian
	i := atomic.AddUint32(&GUIDCounter, 1)
	b[9] = byte(i >> 16)
	b[10] = byte(i >> 8)
	b[11] = byte(i)
	return GUIDType(b[:])
}

func (id GUIDType) Hex() string {
	return hex.EncodeToString([]byte(id))
}

/*
GuuID guuid
*/
func GUID() string {
	objID := NewGUID()
	return objID.Hex()
}
