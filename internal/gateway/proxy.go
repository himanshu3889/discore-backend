package gateway

import (
	"discore/configs"
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// Proxy handler for regular HTTP requests
func (g *Gateway) proxyHandler(target string) gin.HandlerFunc {
	targetURL, _ := url.Parse(target)
	proxy := httputil.NewSingleHostReverseProxy(targetURL)

	// Secure the request
	proxy.Director = func(req *http.Request) {
		req.URL.Scheme = targetURL.Scheme
		req.URL.Host = targetURL.Host
		req.Header.Set("X-Gateway-Secret", configs.Config.GATEWAY_SECRET) // TODO: need to use for the service
	}

	// Custom error handling
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		logrus.WithError(err).Error("Proxy error")
		http.Error(w, "Service unavailable", http.StatusBadGateway)
	}

	return func(c *gin.Context) {
		proxy.ServeHTTP(c.Writer, c.Request)
	}
}
