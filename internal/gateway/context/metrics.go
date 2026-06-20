//revive:disable:package-directory-mismatch
package gatewaycontext

import stdcontext "context"

type upstreamMetricsKey struct{}

// UpstreamMetrics records details about the proxied upstream response.
type UpstreamMetrics struct {
	StatusCode   int
	LatencyMs    int64
	ResponseSize int64
}

// SetUpstreamMetrics stores upstream metrics in the request context.
func SetUpstreamMetrics(ctx stdcontext.Context, metrics UpstreamMetrics) stdcontext.Context {
	return stdcontext.WithValue(ctx, upstreamMetricsKey{}, metrics)
}

// GetUpstreamMetrics reads upstream metrics from the request context.
func GetUpstreamMetrics(ctx stdcontext.Context) UpstreamMetrics {
	metrics, ok := ctx.Value(upstreamMetricsKey{}).(UpstreamMetrics)
	if !ok {
		return UpstreamMetrics{}
	}

	return metrics
}
