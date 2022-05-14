package bsocket

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"net"
	"strconv"
	"sync"
)

var bodyPool = sync.Pool{
	New: func() any {
		return make([]byte, 1024)
	},
}

type BodyRead struct {
	conn net.Conn
}

func (b *BodyRead) Numerical(v any) error {
	return binary.Read(b.conn, binary.BigEndian, v)
}

func (b *BodyRead) Numericals(vs ...any) error {
	for _, v := range vs {
		err := binary.Read(b.conn, binary.BigEndian, v)
		if err != nil {
			return err
		}
	}
	return nil
}

func (b *BodyRead) Body(lng int64) ([]byte, error) {
	body := make([]byte, lng)
	_, err := io.ReadFull(b.conn, body)
	return body, err
}

type BodyWrite struct {
	w *bytes.Buffer
}

func (b *BodyWrite) Numerical(v any) error {
	return binary.Write(b.w, binary.BigEndian, v)
}

func (b *BodyWrite) Numericals(vs ...any) error {
	for _, v := range vs {
		err := binary.Write(b.w, binary.BigEndian, v)
		if err != nil {
			return err
		}
	}
	return nil
}

func (b *BodyWrite) Write(m []byte) (int, error) {
	return b.w.Write(m)
}

func (b *BodyWrite) Bytes() []byte {
	return b.w.Bytes()
}

func (b *BodyWrite) Reset() {
	b.w.Reset()
}

func (b *BodyWrite) Push(conn net.Conn) (int, error) {
	return conn.Write(b.w.Bytes())
}

type SocketConn struct {
	Conn net.Conn
	read *BodyRead
}

func (s *SocketConn) ReadBody() *BodyRead {
	return s.read
}

func (s *SocketConn) WriteBody(b *BodyWrite) (int, error) {
	if b == nil {
		return 0, errors.New("body is empty")
	}
	return s.Conn.Write(b.Bytes())
}

func (s *SocketConn) Close() error {
	return s.Conn.Close()
}

func NewSocketConn(c net.Conn) *SocketConn {
	return &SocketConn{
		Conn: c,
		read: &BodyRead{
			conn: c,
		},
	}
}

func SocketListen(port int, sFunc func(s *SocketConn) error) error {
	srv, err := net.Listen("tcp", "0.0.0.0:"+strconv.FormatInt(int64(port), 10))
	if err != nil {
		return err
	}
	for {
		conn, err := srv.Accept()
		if err != nil {
			return err
		}
		go func(s *SocketConn) {
			defer s.Close()
			for {
				err := sFunc(s)
				if err != nil {
					break
				}
			}
		}(NewSocketConn(conn))
	}
}

var globalSocket = make(map[any]*SocketConn)
var lockSocket sync.RWMutex

func LoadGlobalSocket(size int) {
	globalSocket = make(map[any]*SocketConn, size)
}

func SetGlobalSocket(sid any, s *SocketConn) {
	lockSocket.RLock()
	defer lockSocket.RUnlock()
	globalSocket[sid] = s
}

func GetGlobalSocket(sid any) *SocketConn {
	lockSocket.Lock()
	defer lockSocket.Unlock()
	return globalSocket[sid]
}
