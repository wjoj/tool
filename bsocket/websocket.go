package bsocket

import (
	"net/http"
	"net/url"
	"sync"

	"github.com/gorilla/websocket"
)

type WebsocketConfig struct {
	ReadBufferSize  int
	WriteBufferSize int
	Origins         []string
}

func NewWebsocketUpgrade(cfg *WebsocketConfig) websocket.Upgrader {
	origins := make(map[string]bool, len(cfg.Origins))
	for _, o := range cfg.Origins {
		origins[o] = true
	}

	return websocket.Upgrader{
		ReadBufferSize:  cfg.ReadBufferSize,
		WriteBufferSize: cfg.WriteBufferSize,
		CheckOrigin: func(r *http.Request) bool {
			if len(origins) != 0 {
				origin := r.Header["Origin"]
				if len(origin) == 0 {
					return false
				}
				u, err := url.Parse(origin[0])
				if err != nil {
					return false
				}
				if u.Host == r.Host {
					return true
				}
				return false
			} else {
				return true
			}
		},
	}
}

var defaultUpgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type WbConn struct {
	conn *websocket.Conn
}

func NewWbConn(c *websocket.Conn) *WbConn {
	return &WbConn{
		conn: c,
	}
}

func (c *WbConn) Reset(conn *websocket.Conn) {
	c.conn = conn
}

func (c *WbConn) Conn() *websocket.Conn {
	return c.conn
}

func (c *WbConn) Read() (messageType int, p []byte, err error) {
	return c.conn.ReadMessage()
}

func (c *WbConn) Write(messageType int, data []byte) error {
	return c.conn.WriteMessage(messageType, data)
}

func (c *WbConn) Close() error {
	return c.conn.Close()
}

var wbConnPool = sync.Pool{
	New: func() any {
		return NewWbConn(nil)
	},
}

func getWbConn(conn *websocket.Conn) *WbConn {
	s := wbConnPool.Get().(*WbConn)
	s.Reset(conn)
	return s
}

func releaseWbConn(conn *WbConn) {
	wbConnPool.Put(conn)
}

func WbConnHandle(w http.ResponseWriter, r *http.Request,
	conFunc func(c *WbConn) error,
	bFunc func(msgType int, revbody []byte) (msgId string, sendbody []byte, err error),
	writeErrorFunc func(msgId string, err error)) error {
	conn, err := defaultUpgrader.Upgrade(w, r, nil)
	if err != nil {
		return err
	}
	wbConn := getWbConn(conn)
	defer func() {
		wbConn.Close()
		releaseWbConn(wbConn)
	}()
	err = conFunc(wbConn)
	if err != nil {
		return err
	}
	var er error
	for {
		ty, body, err := wbConn.Read()
		if err != nil {
			er = err
			break
		}
		msgId, sbody, err := bFunc(ty, body)
		if err != nil {
			break
		}
		writeErrorFunc(msgId, wbConn.Write(ty, sbody))
	}
	return er
}

var globalWbConn = make(map[any]*WbConn)
var lockWbConn sync.RWMutex

func LoadGlobalWbConn(size int) {
	globalWbConn = make(map[any]*WbConn, size)
}

func SetGlobalSWbConn(sid any, s *WbConn) {
	lockWbConn.Lock()
	defer lockWbConn.Unlock()
	globalWbConn[sid] = s
}

func GetGlobalWbConn(sid any) *WbConn {
	lockWbConn.RLock()
	defer lockWbConn.RUnlock()
	return globalWbConn[sid]
}
