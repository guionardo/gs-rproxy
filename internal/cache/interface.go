package cache

import (
	"net/http"
	"time"

	"github.com/guionardo/gs-rproxy/internal/httpdata"
)

type Cache interface {
	// If the request is cached, write the response and returns nil
	WriteCachedItem(*http.Request, http.ResponseWriter) error

	SaveCachedItem(*http.Request, *httpdata.HttpResponse) time.Time
}
