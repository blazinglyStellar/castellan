package proxy

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"
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

var (
	errMissingProvider = errors.New("missing provider ID in path")

	defaultTransport = &http.Transport{
		MaxIdleConns:    defaultMaxIdleConns,
		IdleConnTimeout: defaultIdleTimeout,
	}
)

type ProviderResolver interface {
	ResolveBaseURL(ctx context.Context, id string) (string, error)
}

type Proxy struct {
	resolver ProviderResolver
	logger   *slog.Logger
}

func NewReverseProxy(resolver ProviderResolver, logger *slog.Logger) *Proxy {
	return &Proxy{
		resolver: resolver,
		logger:   logger,
	}
}

func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var upstreamStatusCode int
	var upstreamBytes int64
	start := time.Now()

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

			return nil
		},
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			p.errorHandler(w, r, err)
		},
		Transport: defaultTransport,
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
	providerID, restPath, ok := parseProviderPath(outReq.URL.Path)
	if !ok {
		ctx := context.WithValue(outReq.Context(), keyDirectorErr, errMissingProvider)
		*outReq = *outReq.WithContext(ctx)

		return
	}

	baseURL, err := p.resolver.ResolveBaseURL(outReq.Context(), providerID)
	if err != nil {
		p.logger.WarnContext(outReq.Context(),
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
		p.logger.WarnContext(outReq.Context(),
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
}

func (p *Proxy) errorHandler(w http.ResponseWriter, r *http.Request, err error) {
	if ctxErr, ok := r.Context().Value(keyDirectorErr).(error); ok {
		p.logger.WarnContext(r.Context(),
			"provider resolution failed in director",
			slog.String("error", ctxErr.Error()),
		)
		writeJSON(r.Context(), w, http.StatusBadGateway, map[string]string{"error": "invalid provider"})

		return
	}

	p.logger.ErrorContext(r.Context(),
		"upstream request failed",
		slog.String("error", err.Error()),
	)
	writeJSON(r.Context(), w, http.StatusBadGateway, map[string]string{"error": "upstream request failed"})
}

func parseProviderPath(path string) (providerID, rest string, ok bool) {
	const prefix = "/v1/providers/"
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
		slog.ErrorContext(ctx,
			"failed to encode error response",
			slog.String("error", err.Error()),
		)
	}
}
