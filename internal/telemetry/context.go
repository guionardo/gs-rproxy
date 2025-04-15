package telemetry

import (
	"context"
	"net/http"
)

type cntName string

func SetContainer(r *http.Request, containerName string) *http.Request {
	ctx := context.WithValue(r.Context(), cntName("cnt"), containerName)
	return r.WithContext(ctx)
}

func GetContainer(r *http.Request) string {
	containername := r.Context().Value(cntName("cnt")).(string)
	return containername
}
