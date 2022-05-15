package http

import (
	"bufio"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/wjoj/tool/util"
)

func GinProxy(m map[string]string) func(c *gin.Context) {
	rnode := util.NewTrieRootsFromMap(m, "/")
	return func(c *gin.Context) {
		req := c.Request
		node := rnode.InfoExistFromRouting(req.RequestURI)
		if node == nil {
			c.String(http.StatusServiceUnavailable, "domin is empty", 4000)
			return
		}
		httpapi := node.Value.(string)
		if len(httpapi) == 0 {
			c.String(http.StatusServiceUnavailable, "domin is empty", 4001)
			return
		}

		proxy, err := url.Parse(httpapi)
		if err != nil {
			c.String(http.StatusBadGateway, "%v(%v)", err.Error(), 4002)
			return
		}
		req.URL.Scheme = proxy.Scheme
		req.URL.Host = proxy.Host
		_, is := req.Header["If-Modified-Since"]
		if is {
			req.Header.Set("If-Modified-Since", fmt.Sprintf("%v", time.Now().Format("2006-01-02 15:04:05")))
		}
		req.Header.Set("Client-IP", c.ClientIP())
		transport := http.DefaultTransport

		resp, err := transport.RoundTrip(req)
		if err != nil {
			c.String(http.StatusNotImplemented, "%v(%v)", err.Error(), 4003)
			return
		}

		for k, vv := range resp.Header {
			for _, v := range vv {
				c.Header(k, v)
			}
		}
		defer resp.Body.Close()
		c.Status(resp.StatusCode)
		bufio.NewReader(resp.Body).WriteTo(c.Writer)
	}
}
