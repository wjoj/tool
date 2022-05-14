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

type Buffer struct {
	B []byte
}

func newBuffer(size int64) *Buffer {
	return &Buffer{
		B: make([]byte, size),
	}
}

var bodyPool1024 = sync.Pool{
	New: func() any {
		return newBuffer(1024)
	},
}

var bodyPool1024o2 = sync.Pool{
	New: func() any {
		return newBuffer(1024 * 2)
	},
}

var bodyPool1024o3 = sync.Pool{
	New: func() any {
		return newBuffer(1024 * 3)
	},
}

var bodyPool1024o4 = sync.Pool{
	New: func() any {
		return newBuffer(1024 * 4)
	},
}

func getPoolBody(lng int64) *Buffer {
	if lng <= 1024 {
		return bodyPool1024.Get().(*Buffer)
	} else if lng > 1024 && lng <= 1024*2 {
		return bodyPool1024o2.Get().(*Buffer)
	} else if lng > 1024*2 && lng <= 1024*3 {
		return bodyPool1024o3.Get().(*Buffer)
	} else if lng > 1024*3 && lng <= 1024*4 {
		return bodyPool1024o3.Get().(*Buffer)
	} else {
		return newBuffer(lng)
	}
}

func releasePoolBody(b *Buffer) {
	lng := len(b.B)
	if lng <= 1024 {
		bodyPool1024.Put(b)
	} else if lng > 1024 && lng <= 1024*2 {
		bodyPool1024o2.Put(b)
	} else if lng > 1024*2 && lng <= 1024*3 {
		bodyPool1024o3.Put(b)
	} else if lng > 1024*3 && lng <= 1024*4 {
		bodyPool1024o3.Put(b)
	}
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
	body := getPoolBody(lng)
	defer releasePoolBody(body)
	_, err := io.ReadFull(b.conn, body.B[:lng])
	return body.B[:lng], err
}

type BodyWrite struct {
	w *bytes.Buffer
}

func NewBodyWrite() *BodyWrite {
	return &BodyWrite{
		w: new(bytes.Buffer),
	}
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

func (s *SocketConn) Reset(conn net.Conn) {
	s.Conn = conn
	s.read = &BodyRead{
		conn: conn,
	}
}

func NewSocketConn(c net.Conn) *SocketConn {
	return &SocketConn{
		Conn: c,
		read: &BodyRead{
			conn: c,
		},
	}
}

var socketPool = sync.Pool{
	New: func() any {
		return &SocketConn{}
	},
}

func getSocketConn(conn net.Conn) *SocketConn {
	s := socketPool.Get().(*SocketConn)
	s.Reset(conn)
	return s
}

func releaseSocketConn(s *SocketConn) {
	socketPool.Put(s)
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
			defer func() {
				s.Close()
				releaseSocketConn(s)
			}()
			for {
				err := sFunc(s)
				if err != nil {
					break
				}
			}
		}(getSocketConn(conn))
	}
}

func SocketClient(host string, sFunc func(s *SocketConn) error) error {
	conn, err := net.Dial("tcp", host)
	if err != nil {
		return err
	}
	go func(s *SocketConn) {
		defer func() {
			s.Close()
			releaseSocketConn(s)
		}()
		for {
			err := sFunc(s)
			if err != nil {
				break
			}
		}
	}(getSocketConn(conn))

	return nil
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
