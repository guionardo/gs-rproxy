package app

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strings"

	http_errors "github.com/guionardo/gs-rproxy/internal/errors"
)

func getSubdomain(r *http.Request, domain string) string {
	reqUrl, _ := url.Parse("http://" + r.Host)
	hostName := reqUrl.Hostname()

	before, found := strings.CutSuffix(hostName, "."+domain)
	if found {
		return before
	}
	return ""
}

func writeError(w http.ResponseWriter, err error) {
	var statusCode = http.StatusBadGateway
	var responseBody []byte
	switch e := err.(type) {
	case http_errors.HttpError:
		if e.StatusCode > 0 {
			statusCode = e.StatusCode
		}
		responseBody, _ = json.Marshal(e)
	default:
		genErr := http_errors.NewHttpError(err, statusCode)
		responseBody, _ = json.Marshal(genErr)
	}
	w.WriteHeader(statusCode)
	w.Header().Add("Content-Type", "application/json")
	w.Write(responseBody)
}
