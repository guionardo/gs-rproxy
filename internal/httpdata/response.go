package httpdata

import (
	"bytes"
	"io"
	"net/http"
)

type HttpResponse struct {
	StatusCode uint16
	Length     uint64
	Headers    http.Header
	Body       []byte
}

func ResponseFromHttp(r *http.Response) *HttpResponse {
	var body []byte
	if r.Body != nil {
		body, _ = io.ReadAll(r.Body)
	}

	var w = &bytes.Buffer{}
	r.Header.Write(w)
	return &HttpResponse{
		StatusCode: uint16(r.StatusCode),
		Headers:    r.Header.Clone(),
		Body:       body,
		Length:     2 + 8 + uint64(w.Len()) + uint64(len(body)),
	}
}
