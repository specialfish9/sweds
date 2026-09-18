package main

import (
	"crypto/sha256"
	"crypto/subtle"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/specialfish9/confuso/v2"
	"golang.org/x/net/webdav"
)

// Config is loaded from a YAML file via confuso. All fields are optional and
// fall back to sensible defaults when absent.
type Config struct {
	Addr   confuso.Optional[string] `confuso:"addr"`   // listen address (default ":8080")
	Dir    confuso.Optional[string] `confuso:"dir"`    // directory to serve (default "./data")
	Prefix confuso.Optional[string] `confuso:"prefix"` // URL prefix to mount under (default "")
	User   confuso.Optional[string] `confuso:"user"`   // basic auth username (empty disables auth)
	Pass   confuso.Optional[string] `confuso:"pass"`   // basic auth password
	Cert   confuso.Optional[string] `confuso:"cert"`   // TLS certificate file (optional)
	Key    confuso.Optional[string] `confuso:"key"`    // TLS key file (optional)
}

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, nil)))

	path := os.Getenv("CONFIG_PATH")
	if path == "" {
		path = "config.yaml"
	}

	var cfg Config
	if err := confuso.Do(path, &cfg); err != nil {
		slog.Error("cannot load config", "path", path, "err", err)
		os.Exit(1)
	}

	addr := cfg.Addr.Or(":8080")
	dir := cfg.Dir.Or("./data")
	prefix := strings.TrimRight(cfg.Prefix.Or(""), "/")
	user := cfg.User.Or("")
	pass := cfg.Pass.Or("")
	cert := cfg.Cert.Or("")
	key := cfg.Key.Or("")

	if err := os.MkdirAll(dir, 0o755); err != nil {
		slog.Error("cannot create dir", "dir", dir, "err", err)
		os.Exit(1)
	}

	dav := &webdav.Handler{
		Prefix:     prefix,
		FileSystem: webdav.Dir(dir),   // confined to dir; ".." cannot escape it
		LockSystem: webdav.NewMemLS(), // in-memory locks; required by macOS Finder & Office
		Logger: func(r *http.Request, err error) {
			if err != nil {
				slog.Error("webdav", "method", r.Method, "path", r.URL.Path, "err", err)
			} else {
				slog.Info("webdav", "method", r.Method, "path", r.URL.Path)
			}
		},
	}

	var handler http.Handler = dav
	if user != "" {
		handler = basicAuth(dav, user, pass)
	} else {
		slog.Warn("authentication is disabled")
	}

	mux := http.NewServeMux()
	mux.Handle(prefix+"/", handler)

	srv := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       120 * time.Second,
		// No Read/WriteTimeout: they would abort large uploads/downloads.
	}

	slog.Info("serving", "dir", dir, "addr", addr, "prefix", prefix)
	if cert != "" && key != "" {
		slog.Error("server stopped", "err", srv.ListenAndServeTLS(cert, key))
		os.Exit(1)
	}
	slog.Error("server stopped", "err", srv.ListenAndServe())
	os.Exit(1)
}

// basicAuth wraps next with HTTP Basic authentication using constant-time comparison.
func basicAuth(next http.Handler, user, pass string) http.Handler {
	// Hash first so ConstantTimeCompare sees equal-length inputs
	// and doesn't leak length information.
	wantU := sha256.Sum256([]byte(user))
	wantP := sha256.Sum256([]byte(pass))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u, p, ok := r.BasicAuth()
		if ok {
			gotU := sha256.Sum256([]byte(u))
			gotP := sha256.Sum256([]byte(p))
			userOK := subtle.ConstantTimeCompare(gotU[:], wantU[:])
			passOK := subtle.ConstantTimeCompare(gotP[:], wantP[:])
			if userOK&passOK == 1 {
				next.ServeHTTP(w, r)
				return
			}
		}
		w.Header().Set("WWW-Authenticate", `Basic realm="webdav", charset="UTF-8"`)
		http.Error(w, "unauthorized", http.StatusUnauthorized)
	})
}
