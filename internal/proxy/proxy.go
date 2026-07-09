package proxy

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"strings"
	"time"

	gatewaycontext "castellan/internal/gateway/context"
	"castellan/internal/server/middleware"
)

type contextKey int

const (
	keyDirectorErr contextKey = iota
)

const (
	defaultMaxIdleConns = 100
	defaultIdleTimeout  = 90 * time.Second
)

var errMissingProvider = errors.New("missing provider ID in path")

type timeoutConn struct {
	net.Conn
	readTimeout  time.Duration
	writeTimeout time.Duration
}

func (c *timeoutConn) Read(b []byte) (int, error) {
	if c.readTimeout > 0 {
		if err := c.SetReadDeadline(time.Now().Add(c.readTimeout)); err != nil {
			return 0, fmt.Errorf("set read deadline: %w", err)
		}
	}
	n, err := c.Conn.Read(b)
	if err != nil {
		return n, fmt.Errorf("read from upstream: %w", err)
	}
	return n, nil
}

func (c *timeoutConn) Write(b []byte) (int, error) {
	if c.writeTimeout > 0 {
		if err := c.SetWriteDeadline(time.Now().Add(c.writeTimeout)); err != nil {
			return 0, fmt.Errorf("set write deadline: %w", err)
		}
	}
	n, err := c.Conn.Write(b)
	if err != nil {
		return n, fmt.Errorf("write to upstream: %w", err)
	}
	return n, nil
}

// ProviderResolver resolves an upstream base URL for a given provider ID.
type ProviderResolver interface {
	ResolveBaseURL(ctx context.Context, id string) (string, error)
}

// Proxy is a reverse proxy that resolves upstream URLs, applies timeouts,
// and retries failed requests with jittered exponential backoff.
type Proxy struct {
	resolver    ProviderResolver
	logger      *slog.Logger
	cfg         Config
	transport   *http.Transport
	retryPolicy RetryPolicy
}

// NewReverseProxy creates a Proxy with a custom transport that enforces
// connect/read/write timeouts and the given retry policy.
func NewReverseProxy(resolver ProviderResolver, logger *slog.Logger, cfg Config) *Proxy {
	dialer := &net.Dialer{Timeout: cfg.ConnectTimeout}

	baseTransport, ok := http.DefaultTransport.(*http.Transport)
	if !ok {
		baseTransport = &http.Transport{}
	} else {
		baseTransport = baseTransport.Clone()
	}

	baseTransport.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
		conn, err := dialer.DialContext(ctx, network, addr)
		if err != nil {
			return nil, fmt.Errorf("dial upstream: %w", err)
		}
		return &timeoutConn{
			Conn:         conn,
			readTimeout:  cfg.ReadTimeout,
			writeTimeout: cfg.WriteTimeout,
		}, nil
	}
	baseTransport.ResponseHeaderTimeout = cfg.ReadTimeout
	baseTransport.MaxIdleConns = defaultMaxIdleConns
	baseTransport.IdleConnTimeout = defaultIdleTimeout

	retryPolicy := DefaultRetryPolicy()
	if cfg.RetryMaxRetries > 0 {
		retryPolicy = RetryPolicy{
			MaxRetries: cfg.RetryMaxRetries,
			BaseDelay:  cfg.RetryBaseDelay,
			MaxDelay:   cfg.RetryMaxDelay,
		}
	}

	return &Proxy{
		resolver:    resolver,
		logger:      logger,
		cfg:         cfg,
		transport:   baseTransport,
		retryPolicy: retryPolicy,
	}
}

type retryRoundTripper struct {
	inner  http.RoundTripper
	policy RetryPolicy
	logger *slog.Logger
}

func (r *retryRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	var bodyBytes []byte

	if req.Body != nil {
		var err error
		bodyBytes, err = io.ReadAll(req.Body)
		if err != nil {
			return nil, fmt.Errorf("cache request body for retry: %w", err)
		}
		// #nosec G104 - closing original body after buffering; error not actionable
		req.Body.Close()
		req.Body = nil
	}

	if bodyBytes != nil {
		buf := bodyBytes
		req.GetBody = func() (io.ReadCloser, error) {
			return io.NopCloser(bytes.NewReader(buf)), nil
		}
	}

	var lastResp *http.Response
	var lastErr error

	for attempt := 0; attempt <= r.policy.MaxRetries; attempt++ {
		if attempt > 0 {
			delay := r.policy.Backoff(attempt - 1)

			r.logger.WarnContext(
				req.Context(),
				"retrying upstream request",
				slog.Int("attempt", attempt),
				slog.String("method", req.Method),
				slog.String("url", req.URL.String()),
				slog.String("delay", delay.String()),
			)

			select {
			case <-req.Context().Done():
				return nil, fmt.Errorf("retry cancelled: %w", req.Context().Err())
			case <-time.After(delay):
			}

			if req.GetBody != nil {
				body, err := req.GetBody()
				if err != nil {
					return nil, fmt.Errorf("get request body for retry: %w", err)
				}
				req.Body = body
			}
		}

		resp, err := r.inner.RoundTrip(req)
		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return resp, fmt.Errorf("retry aborted: %w", err)
			}

			if !isIdempotent(req.Method) || attempt >= r.policy.MaxRetries {
				return resp, fmt.Errorf("upstream round trip: %w", err)
			}

			lastResp = resp
			lastErr = err

			continue
		}

		if !r.policy.ShouldRetry(resp.StatusCode, nil) {
			return resp, nil
		}

		if !isIdempotent(req.Method) || attempt >= r.policy.MaxRetries {
			return resp, nil
		}

		// #nosec G104 - closing discarded intermediate retry response; error not actionable
		resp.Body.Close()
		lastResp = resp
		lastErr = nil

		continue
	}

	return lastResp, lastErr
}

// ServeHTTP forwards the request to the upstream provider and records upstream metrics on the context.
func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var upstreamStatusCode int
	var upstreamBytes int64
	start := time.Now()
	requestStart := gatewaycontext.GetRequestStart(r.Context())

	rp := &httputil.ReverseProxy{
		Director: func(outReq *http.Request) {
			p.director(outReq)
		},
		ModifyResponse: func(resp *http.Response) error {
			upstreamStatusCode = resp.StatusCode
			resp.Body = &countingReadCloser{
				ReadCloser: resp.Body,
				counter:    &upstreamBytes,
			}

			if !requestStart.IsZero() {
				preproxyMs := time.Since(requestStart).Milliseconds()
				upstreamMs := time.Since(start).Milliseconds()
				resp.Header.Set("X-Castellan-Preproxy-Ms", strconv.FormatInt(preproxyMs, 10))
				resp.Header.Set("X-Castellan-Upstream-Ms", strconv.FormatInt(upstreamMs, 10))
			}

			return nil
		},
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			p.errorHandler(w, r, err)
		},
		Transport: &retryRoundTripper{
			inner:  p.transport,
			policy: p.retryPolicy,
			logger: p.logger,
		},
	}

	rp.ServeHTTP(w, r)

	ctx := gatewaycontext.SetUpstreamMetrics(r.Context(), gatewaycontext.UpstreamMetrics{
		StatusCode:   upstreamStatusCode,
		LatencyMs:    time.Since(start).Milliseconds(),
		ResponseSize: upstreamBytes,
	})
	*r = *r.WithContext(ctx)
}

func (p *Proxy) director(outReq *http.Request) {
	providerID, restPath, ok := ParseProviderPath(outReq.URL.Path)
	if !ok {
		ctx := context.WithValue(outReq.Context(), keyDirectorErr, errMissingProvider)
		*outReq = *outReq.WithContext(ctx)

		return
	}

	baseURL, err := p.resolver.ResolveBaseURL(outReq.Context(), providerID)
	if err != nil {
		p.logger.WarnContext(
			outReq.Context(),
			"provider resolution failed",
			slog.String("provider_id", providerID),
			slog.String("error", err.Error()),
		)
		ctx := context.WithValue(outReq.Context(), keyDirectorErr, err)
		*outReq = *outReq.WithContext(ctx)

		return
	}

	target, err := url.Parse(baseURL + restPath)
	if err != nil {
		p.logger.WarnContext(
			outReq.Context(),
			"invalid target url",
			slog.String("base_url", baseURL),
			slog.String("rest_path", restPath),
			slog.String("error", err.Error()),
		)
		ctx := context.WithValue(outReq.Context(), keyDirectorErr, err)
		*outReq = *outReq.WithContext(ctx)

		return
	}

	savedQuery := outReq.URL.RawQuery
	outReq.URL = target
	outReq.Host = target.Host
	if savedQuery != "" {
		outReq.URL.RawQuery = savedQuery
	}

	outReq.Header.Set("X-Forwarded-For", outReq.RemoteAddr)
	outReq.Header.Set("X-Request-ID", middleware.GetRequestID(outReq.Context()))

	consumer := gatewaycontext.GetConsumerInfo(outReq.Context())
	if consumer.ConsumerID != "" {
		outReq.Header.Set("X-Castellan-Consumer", consumer.ConsumerID)
	}

	outReq.Header.Del("Authorization")
}

func (p *Proxy) errorHandler(w http.ResponseWriter, r *http.Request, err error) {
	if ctxErr, ok := r.Context().Value(keyDirectorErr).(error); ok {
		p.logger.WarnContext(
			r.Context(),
			"provider resolution failed in director",
			slog.String("error", ctxErr.Error()),
		)
		writeJSON(r.Context(), w, http.StatusBadGateway, map[string]string{"error": "invalid provider"})

		return
	}

	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		p.logger.ErrorContext(
			r.Context(),
			"upstream request timed out",
			slog.String("error", err.Error()),
		)
		writeJSON(r.Context(), w, http.StatusGatewayTimeout, map[string]string{"error": "upstream request timed out"})

		return
	}

	p.logger.ErrorContext(
		r.Context(),
		"upstream request failed",
		slog.String("error", err.Error()),
	)
	writeJSON(r.Context(), w, http.StatusBadGateway, map[string]string{"error": "upstream request failed"})
}

func ParseProviderPath(path string) (providerID, rest string, ok bool) {
	const prefix = "/api/gateway/"
	const splitParts = 2

	if !strings.HasPrefix(path, prefix) {
		return "", "", false
	}

	trimmed := strings.TrimPrefix(path, prefix)
	parts := strings.SplitN(trimmed, "/", splitParts)
	if len(parts) == 0 || parts[0] == "" {
		return "", "", false
	}

	providerID = parts[0]

	if len(parts) > 1 {
		rest = "/" + parts[1]
	} else {
		rest = "/"
	}

	return providerID, rest, true
}

func writeJSON(ctx context.Context, w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.ErrorContext(
			ctx,
			"failed to encode error response",
			slog.String("error", err.Error()),
		)
	}
}
