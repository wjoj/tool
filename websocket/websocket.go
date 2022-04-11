package websocket

import (
	"net/http"
	"net/url"

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

func WbConnRead(w http.ResponseWriter, r *http.Request,
	conFunc func(c *websocket.Conn) error,
	bFunc func(msgType int, revbody []byte) (msgId string, sendbody []byte, err error),
	writeErrorFunc func(msgId string, err error)) error {
	conn, err := defaultUpgrader.Upgrade(w, r, nil)
	if err != nil {
		return err
	}
	err = conFunc(conn)
	if err != nil {
		conn.Close()
		return err
	}
	var er error
	for {
		ty, body, err := conn.ReadMessage()
		if err != nil {
			er = err
			break
		}
		msgId, sbody, err := bFunc(ty, body)
		if err != nil {
			conn.Close()
			break
		}
		writeErrorFunc(msgId, conn.WriteMessage(ty, sbody))
	}
	return er
}
