// Package facade implements the Unix-socket HTTP reverse proxy.
//
// Routing: requests whose path is /<mount>/<rest> are forwarded to
//
//	http://127.0.0.1:<port>/<rest>
//
// Streaming is handled by wrapping http.ResponseWriter with an
// immediate-flush writer: chunked responses and text/event-stream
// pass through without buffering.
//
// Upgrades (WebSocket) are handled via hijacking: after proxying the
// handshake, raw TCP bytes are spliced bidirectionally.
package facade

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Route maps a mount prefix to an upstream localhost port.
type Route struct {
	Mount        string // e.g. "slack"
	UpstreamPort int
	MaxBodyBytes int64         // 0 = inherit facade default
	IdleTimeout  time.Duration // 0 = inherit facade default
	Skill        string        // registry name
}

// Facade is a Unix-socket HTTP server that reverse-proxies to registered upstreams.
type Facade struct {
	SocketPath    string
	Routes        []Route
	MaxBodyBytes  int64
	IdleTimeout   time.Duration
	AccessLogPath string
	Version       string

	mu      sync.RWMutex
	routes  map[string]*Route
	server  *http.Server
	listen  net.Listener
	accLog  *log.Logger
	accFile *os.File
}

// New constructs a Facade from a slice of Routes.
func New(socketPath string, routes []Route, maxBody int64, idle time.Duration, accessLog, version string) *Facade {
	m := make(map[string]*Route, len(routes))
	for i := range routes {
		r := routes[i]
		m[r.Mount] = &r
	}
	return &Facade{
		SocketPath:    socketPath,
		Routes:        routes,
		MaxBodyBytes:  maxBody,
		IdleTimeout:   idle,
		AccessLogPath: accessLog,
		Version:       version,
		routes:        m,
	}
}

// Start opens the socket and begins serving. Returns once the listener is bound.
// Call Close to stop.
func (f *Facade) Start(ctx context.Context) error {
	_ = os.Remove(f.SocketPath) // clean stale
	if err := os.MkdirAll(dirOf(f.SocketPath), 0o700); err != nil {
		return fmt.Errorf("facade: mkdir socket dir: %w", err)
	}
	ln, err := net.Listen("unix", f.SocketPath)
	if err != nil {
		return fmt.Errorf("facade: listen unix %s: %w", f.SocketPath, err)
	}
	if err := os.Chmod(f.SocketPath, 0o600); err != nil {
		ln.Close()
		return fmt.Errorf("facade: chmod socket: %w", err)
	}
	if f.AccessLogPath != "" {
		af, err := os.OpenFile(f.AccessLogPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
		if err != nil {
			ln.Close()
			return fmt.Errorf("facade: open access log: %w", err)
		}
		f.accFile = af
		f.accLog = log.New(af, "", 0)
	}
	f.listen = ln
	mux := http.NewServeMux()
	mux.HandleFunc("/", f.handle)
	f.server = &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: 15 * time.Second,
		IdleTimeout:       f.IdleTimeout,
	}
	go func() {
		if err := f.server.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
			_, _ = fmt.Fprintln(os.Stderr, "facade: serve:", err)
		}
	}()
	return nil
}

// Close tears down the server and removes the socket.
func (f *Facade) Close() error {
	var firstErr error
	if f.server != nil {
		shutCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		if err := f.server.Shutdown(shutCtx); err != nil {
			firstErr = err
		}
	}
	if f.listen != nil {
		_ = f.listen.Close()
	}
	if f.accFile != nil {
		_ = f.accFile.Close()
	}
	if f.SocketPath != "" {
		_ = os.Remove(f.SocketPath)
	}
	return firstErr
}

// handle is the root HTTP handler.
func (f *Facade) handle(w http.ResponseWriter, r *http.Request) {
	started := time.Now()
	if r.URL.Path == "/" || r.URL.Path == "" {
		f.writeStatus(w, r)
		return
	}
	mount, rest := splitMount(r.URL.Path)
	if mount == "" {
		http.Error(w, "omac: invalid path", http.StatusNotFound)
		return
	}
	f.mu.RLock()
	route, ok := f.routes[mount]
	f.mu.RUnlock()
	if !ok {
		w.Header().Set("X-Omac-Reason", "unknown-mount")
		http.Error(w, fmt.Sprintf("omac: unknown skill mount %q", mount), http.StatusNotFound)
		return
	}
	// WebSocket / generic Upgrade path.
	if isUpgrade(r) {
		f.proxyUpgrade(w, r, route, rest, started)
		return
	}
	f.proxyHTTP(w, r, route, rest, started)
}

func (f *Facade) writeStatus(w http.ResponseWriter, _ *http.Request) {
	type status struct {
		Version string   `json:"version"`
		Skills  []string `json:"skills"`
	}
	out := status{Version: f.Version}
	f.mu.RLock()
	for _, r := range f.routes {
		out.Skills = append(out.Skills, r.Mount)
	}
	f.mu.RUnlock()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(out)
}

// proxyHTTP forwards plain HTTP (including SSE).
func (f *Facade) proxyHTTP(w http.ResponseWriter, r *http.Request, route *Route, rest string, started time.Time) {
	upstream := &url.URL{Scheme: "http", Host: upstreamHost(route.UpstreamPort)}
	rp := httputil.NewSingleHostReverseProxy(upstream)

	// Customize the Director so we can rewrite the path and header set.
	rp.Director = func(req *http.Request) {
		req.URL.Scheme = "http"
		req.URL.Host = upstream.Host
		req.URL.Path = "/" + rest
		req.Host = upstream.Host
		req.Header.Set("X-Forwarded-Prefix", "/"+route.Mount)
		// Hop-by-hop headers are stripped by httputil automatically.
	}

	// Enforce max body bytes for inbound request body (best-effort; SSE has no body).
	limit := route.MaxBodyBytes
	if limit == 0 {
		limit = f.MaxBodyBytes
	}
	if limit > 0 && r.Body != nil {
		r.Body = http.MaxBytesReader(w, r.Body, limit)
	}

	// Modify response to ensure SSE isn't buffered by any intermediate.
	rp.ModifyResponse = func(resp *http.Response) error {
		if ct := resp.Header.Get("Content-Type"); strings.HasPrefix(ct, "text/event-stream") {
			// net/http on the server side auto-flushes for chunked, but setting
			// X-Accel-Buffering tells any downstream-minded client we're streaming.
			resp.Header.Set("X-Accel-Buffering", "no")
		}
		return nil
	}

	rp.ErrorHandler = func(w http.ResponseWriter, _ *http.Request, err error) {
		reason := "upstream-error"
		code := http.StatusBadGateway
		if isTimeout(err) {
			reason = "timeout"
			code = http.StatusGatewayTimeout
		} else if isConnRefused(err) {
			reason = "sidecar-down"
			code = http.StatusServiceUnavailable
		}
		w.Header().Set("X-Omac-Reason", reason)
		http.Error(w, "omac: "+reason, code)
	}

	// Short connect timeout; liberal overall IO.
	rp.Transport = &http.Transport{
		DialContext: (&net.Dialer{Timeout: 2 * time.Second}).DialContext,
		// Disable response buffering so SSE frames flush immediately.
		DisableCompression:    true,
		ResponseHeaderTimeout: 30 * time.Second,
	}

	// Wrap ResponseWriter so we capture the upstream status for logging.
	wr := &statusCaptureWriter{ResponseWriter: w, status: http.StatusOK}
	rp.ServeHTTP(wr, r)
	f.logAccess(r, route, rest, wr.status, wr.bytes, time.Since(started))
}

// proxyUpgrade handles HTTP Upgrade requests (WebSocket) by splicing the
// underlying TCP connections after forwarding the handshake.
func (f *Facade) proxyUpgrade(w http.ResponseWriter, r *http.Request, route *Route, rest string, started time.Time) {
	upstreamAddr := upstreamHost(route.UpstreamPort)
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	upConn, err := (&net.Dialer{}).DialContext(ctx, "tcp", upstreamAddr)
	if err != nil {
		w.Header().Set("X-Omac-Reason", "sidecar-down")
		http.Error(w, "omac: upstream dial: "+err.Error(), http.StatusBadGateway)
		return
	}
	defer upConn.Close()

	// Write the rewritten request to the upstream connection.
	clone := r.Clone(r.Context())
	clone.URL = &url.URL{Scheme: "http", Host: upstreamAddr, Path: "/" + rest, RawQuery: r.URL.RawQuery}
	clone.Host = upstreamAddr
	clone.RequestURI = clone.URL.RequestURI()
	clone.Header = r.Header.Clone()
	clone.Header.Set("X-Forwarded-Prefix", "/"+route.Mount)
	if err := clone.Write(upConn); err != nil {
		w.Header().Set("X-Omac-Reason", "upstream-error")
		http.Error(w, "omac: write upstream: "+err.Error(), http.StatusBadGateway)
		return
	}

	// Hijack the client connection.
	hj, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "omac: hijack unsupported", http.StatusInternalServerError)
		return
	}
	clientConn, buf, err := hj.Hijack()
	if err != nil {
		http.Error(w, "omac: hijack: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer clientConn.Close()

	// Flush anything already buffered from the client to the upstream.
	if buf != nil && buf.Reader.Buffered() > 0 {
		if _, err := io.CopyN(upConn, buf.Reader, int64(buf.Reader.Buffered())); err != nil {
			return
		}
	}

	// Splice both directions.
	done := make(chan struct{}, 2)
	go func() { _, _ = io.Copy(upConn, clientConn); done <- struct{}{} }()
	go func() { _, _ = io.Copy(clientConn, upConn); done <- struct{}{} }()
	<-done
	f.logAccess(r, route, rest, 101, 0, time.Since(started))
}

func (f *Facade) logAccess(r *http.Request, route *Route, rest string, status int, bytes int64, dur time.Duration) {
	if f.accLog == nil {
		return
	}
	entry := map[string]any{
		"ts":              time.Now().UTC().Format(time.RFC3339Nano),
		"method":          r.Method,
		"mount":           route.Mount,
		"path":            "/" + rest,
		"upstream_status": status,
		"bytes_out":       bytes,
		"duration_ms":     dur.Milliseconds(),
	}
	b, _ := json.Marshal(entry)
	f.accLog.Println(string(b))
}

// splitMount returns (mount, rest) for a request path.
// "/slack/foo/bar" → ("slack", "foo/bar").
// "/slack/"        → ("slack", "").
// "/slack"         → ("slack", "").
// "/"              → ("", "").
func splitMount(p string) (string, string) {
	if len(p) < 2 || p[0] != '/' {
		return "", ""
	}
	rest := p[1:]
	slash := strings.IndexByte(rest, '/')
	if slash < 0 {
		return rest, ""
	}
	return rest[:slash], rest[slash+1:]
}

func upstreamHost(port int) string { return "127.0.0.1:" + strconv.Itoa(port) }

func isUpgrade(r *http.Request) bool {
	if !headerHasToken(r.Header.Get("Connection"), "upgrade") {
		return false
	}
	return r.Header.Get("Upgrade") != ""
}

func headerHasToken(hdr, token string) bool {
	for _, f := range strings.Split(hdr, ",") {
		if strings.EqualFold(strings.TrimSpace(f), token) {
			return true
		}
	}
	return false
}

func isTimeout(err error) bool {
	var ne net.Error
	return errors.As(err, &ne) && ne.Timeout()
}

func isConnRefused(err error) bool {
	if err == nil {
		return false
	}
	s := err.Error()
	return strings.Contains(s, "connection refused") || strings.Contains(s, "connect: connection refused")
}

func dirOf(p string) string {
	if i := strings.LastIndexByte(p, '/'); i >= 0 {
		return p[:i]
	}
	return "."
}

// statusCaptureWriter records status + byte counts for access logging.
type statusCaptureWriter struct {
	http.ResponseWriter
	status    int
	bytes     int64
	headerSet bool
}

func (w *statusCaptureWriter) WriteHeader(code int) {
	w.status = code
	w.headerSet = true
	w.ResponseWriter.WriteHeader(code)
}

func (w *statusCaptureWriter) Write(b []byte) (int, error) {
	if !w.headerSet {
		w.headerSet = true
	}
	n, err := w.ResponseWriter.Write(b)
	w.bytes += int64(n)
	return n, err
}

// Flush implements http.Flusher when the underlying writer does.
func (w *statusCaptureWriter) Flush() {
	if fl, ok := w.ResponseWriter.(http.Flusher); ok {
		fl.Flush()
	}
}

// (No Hijack method on the wrapper; the facade calls Hijack on the raw
// ResponseWriter in proxyUpgrade before wrapping would have happened.)
