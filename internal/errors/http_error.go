package http_errors

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

type HttpError struct {
	err        error
	StatusCode int    `json:"status_code"`
	Message    string `json:"message,omitempty"`
	ErrorMsg   string `json:"error,omitempty"`
}

func (e HttpError) Error() string {
	sb := strings.Builder{}
	sb.WriteString(strconv.Itoa(e.StatusCode))
	if len(e.Message) > 0 {
		sb.WriteString(" " + e.Message)
	}
	if len(e.ErrorMsg) > 0 {
		sb.WriteString(" " + e.ErrorMsg)
	}
	return sb.String()
}

func NewHttpError(err any, statusCode ...int) HttpError {
	var internalError error
	switch et := err.(type) {
	case error:
		internalError = et
	case fmt.Stringer:
		internalError = errors.New(et.String())
	default:
		internalError = fmt.Errorf("%v", err)

	}
	var sc = http.StatusBadGateway
	if len(statusCode) > 0 {
		sc = statusCode[0]
	}
	return HttpError{
		err:        internalError,
		StatusCode: sc,
		ErrorMsg:   internalError.Error(),
	}
}
