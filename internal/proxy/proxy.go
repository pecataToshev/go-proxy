package proxy

import (
	"crypto/tls"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"proxy/config"
	"strings"
	"sync"
	"time"
)

var (
	transport   *http.Transport
	copyBufPool = sync.Pool{
		New: func() any {
			b := make([]byte, 4096) // 4 KB streaming buffer
			return &b
		},
	}
	proxySem       chan struct{}
	allowedOrigins []string
)

// Hop-by-hop headers that MUST NOT be forwarded.
var hopHeaders = [...]string{
	"Connection",
	"Keep-Alive",
	"Proxy-Authenticate",
	"Proxy-Authorization",
	"Te",
	"Trailers",
	"Transfer-Encoding",
	"Upgrade",
}

// Init initializes the proxy with the provided configuration.
func Init(cfg *config.Config) {
	allowedOrigins = cfg.CORS.AllowedOrigins
	if len(allowedOrigins) > 0 {
		isAllAllowed := false
		for _, origin := range allowedOrigins {
			if origin == "*" {
				isAllAllowed = true
				break
			}
		}

		if isAllAllowed {
			log.Println("CORS: allowing all origins (*)")
		} else {
			log.Printf("CORS: allowing origins: %v", allowedOrigins)
		}
	} else {
		log.Println("CORS: disabled (no allowed_origins in config)")
	}

	transport = &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   time.Duration(cfg.Transport.DialTimeout) * time.Second,
			KeepAlive: time.Duration(cfg.Transport.DialKeepAlive) * time.Second,
		}).DialContext,
		TLSClientConfig:       &tls.Config{MinVersion: tls.VersionTLS12},
		MaxIdleConns:          cfg.Transport.MaxIdleConns,
		MaxIdleConnsPerHost:   cfg.Transport.MaxIdleConnsPerHost,
		MaxConnsPerHost:       cfg.Transport.MaxConnsPerHost,
		IdleConnTimeout:       time.Duration(cfg.Transport.IdleConnTimeout) * time.Second,
		ResponseHeaderTimeout: time.Duration(cfg.Transport.ResponseHeaderTimeout) * time.Second,
		DisableCompression:    true, // pass-through; never decompress
		ReadBufferSize:        cfg.Transport.ReadBufferSize,
		WriteBufferSize:       cfg.Transport.WriteBufferSize,
		ForceAttemptHTTP2:     false, // HTTP/1.1 uses less memory
	}

	proxySem = make(chan struct{}, cfg.Proxy.MaxConcurrentRequests)
}

// CORSMiddleware wraps an http.Handler with CORS logic.
func CORSMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")

		if len(allowedOrigins) > 0 && origin != "" {
			allowed := false

			if allowedOrigins[0] == "*" {
				w.Header().Set("Access-Control-Allow-Origin", "*")
				allowed = true
			} else {
				for _, ao := range allowedOrigins {
					if ao == origin {
						w.Header().Set("Access-Control-Allow-Origin", origin)
						w.Header().Set("Vary", "Origin")
						allowed = true
						break
					}
				}
			}

			if allowed {
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS, PATCH")
				w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")
				w.Header().Set("Access-Control-Max-Age", "86400")

				if r.Method == "OPTIONS" {
					w.WriteHeader(http.StatusNoContent)
					return
				}
			}
		}

		next.ServeHTTP(w, r)
	})
}

// Handler returns an http.HandlerFunc that proxies requests to the target URL.
func Handler(prefix string, target *url.URL) http.HandlerFunc {
	strip := strings.TrimSuffix(prefix, "/")

	return func(w http.ResponseWriter, r *http.Request) {
		select {
		case proxySem <- struct{}{}:
			defer func() { <-proxySem }()
		case <-r.Context().Done():
			http.Error(w, "Service Unavailable", http.StatusServiceUnavailable)
			return
		}

		tail := strings.TrimPrefix(r.URL.Path, strip)
		if tail == "" {
			tail = "/"
		}
		u := url.URL{
			Scheme:   target.Scheme,
			Host:     target.Host,
			Path:     JoinPath(target.Path, tail),
			RawQuery: r.URL.RawQuery,
		}

		out, err := http.NewRequestWithContext(
			r.Context(), r.Method, u.String(), r.Body)
		if err != nil {
			http.Error(w, "Bad Gateway", http.StatusBadGateway)
			return
		}
		out.ContentLength = r.ContentLength

		for k, vv := range r.Header {
			out.Header[k] = vv
		}
		DelHop(out.Header)
		out.Host = target.Host
		out.Header.Set("Host", target.Host)

		if ip, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
			if xff := out.Header.Get("X-Forwarded-For"); xff != "" {
				out.Header.Set("X-Forwarded-For", xff+", "+ip)
			} else {
				out.Header.Set("X-Forwarded-For", ip)
			}
		}

		resp, err := transport.RoundTrip(out)
		if err != nil {
			http.Error(w, "Bad Gateway", http.StatusBadGateway)
			return
		}
		defer resp.Body.Close()

		wh := w.Header()
		corsHeaders := make(map[string][]string)
		for k, vv := range wh {
			if strings.HasPrefix(strings.ToLower(k), "access-control-") || strings.ToLower(k) == "vary" {
				corsHeaders[k] = vv
			}
		}

		for k, vv := range resp.Header {
			wh[k] = vv
		}
		DelHop(wh)

		for k, vv := range corsHeaders {
			wh[k] = vv
		}

		w.WriteHeader(resp.StatusCode)
		bp := copyBufPool.Get().(*[]byte)
		io.CopyBuffer(w, resp.Body, *bp)
		copyBufPool.Put(bp)
	}
}

// DelHop removes hop-by-hop headers.
func DelHop(h http.Header) {
	for _, k := range hopHeaders {
		h.Del(k)
	}
}

// JoinPath joins two paths accurately.
func JoinPath(base, extra string) string {
	if base == "" || base == "/" {
		return extra
	}
	a, b := strings.HasSuffix(base, "/"), strings.HasPrefix(extra, "/")
	switch {
	case a && b:
		return base + extra[1:]
	case !a && !b:
		return base + "/" + extra
	default:
		return base + extra
	}
}
