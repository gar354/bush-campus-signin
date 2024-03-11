package middleware

import (
	"net/http"
	"strings"
)

var (
	xForwardedFor   = http.CanonicalHeaderKey("X-Forwarded-For")
	xForwardedHost  = http.CanonicalHeaderKey("X-Forwarded-Host")
	xForwardedProto = http.CanonicalHeaderKey("X-Forwarded-Proto")
)

type ProxyHandler struct {
	handler http.Handler
}

func NewProxyHandler(handlerToWrap http.Handler) *ProxyHandler {
	return &ProxyHandler{handlerToWrap}
}

func (p *ProxyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Set the remote IP using
	if fwd := r.Header.Get(xForwardedFor); fwd != "" {
		// Only grab the first (client) address.
		s := strings.Index(fwd, ", ")
		if s == -1 {
			s = len(fwd)
		}
		addr := fwd[:s]
		r.RemoteAddr = addr
	}

	// Set the protocol with value forwarded from the proxy.
	if proto := r.Header.Get(xForwardedProto); proto != "" {
		r.URL.Scheme = strings.ToLower(proto)
	}
	// Set the host with the value forwarded by the proxy
	if host := r.Header.Get(xForwardedHost); host != "" {
		r.Host = host
	}
	// Call the next handler in the chain.
	p.handler.ServeHTTP(w, r)
}
