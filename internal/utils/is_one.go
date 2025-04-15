package utils

import (
	"net/http"
	"slices"
)

func IsOneOf[T comparable](value T, options ...T) bool {
	return slices.Contains(options, value)
}

var traceIdHeaders = []string{"X-TraceId", "trace-id"}

func GetTraceId(r *http.Request) string {
	for _, h := range traceIdHeaders {
		if v := r.Header.Get(h); len(v) > 0 {
			return v
		}
	}
	return ""

}
