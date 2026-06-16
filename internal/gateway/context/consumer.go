//revive:disable:package-directory-mismatch
package gatewaycontext

import stdcontext "context"

type consumerInfoKey struct{}

// ConsumerInfo describes the authenticated consumer for a gateway request.
type ConsumerInfo struct {
	ConsumerID      string
	APIKeyID        string
	IsAuthenticated bool
}

// SetConsumerInfo stores consumer information in the request context.
func SetConsumerInfo(ctx stdcontext.Context, info ConsumerInfo) stdcontext.Context {
	return stdcontext.WithValue(ctx, consumerInfoKey{}, info)
}

// GetConsumerInfo reads consumer information from the request context.
func GetConsumerInfo(ctx stdcontext.Context) ConsumerInfo {
	info, ok := ctx.Value(consumerInfoKey{}).(ConsumerInfo)
	if !ok {
		return ConsumerInfo{}
	}

	return info
}
