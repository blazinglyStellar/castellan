//revive:disable:package-directory-mismatch
package gatewaycontext

import (
	stdcontext "context"
	"time"
)

type requestStartKey struct{}

func SetRequestStart(ctx stdcontext.Context, t time.Time) stdcontext.Context {
	return stdcontext.WithValue(ctx, requestStartKey{}, t)
}

func GetRequestStart(ctx stdcontext.Context) time.Time {
	t, ok := ctx.Value(requestStartKey{}).(time.Time)
	if !ok {
		return time.Time{}
	}
	return t
}
