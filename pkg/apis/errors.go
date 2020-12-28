package apis

import (
	"fmt"
	"net/http"

	"github.com/pkg/errors"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type Err struct {
	Message string      `json:"message"`
	Code    string      `json:"code,omitempty"`
	Status  int         `json:"status"`
	Data    interface{} `json:"data,omitempty"`
	Reason  string      `json:"reason,omitempty"`
	Cause   error       `json:"-"`
	Errors  interface{} `json:"errors,omitempty"` // used for form error reporting
}

func (e Err) Error() string {
	return e.Message
}
func FromError(err interface{}) *Err {
	if err == nil {
		return nil
	}
	var e *Err
	switch v := err.(type) {
	case Err:
		e = &v
	case *Err:
		e = v
	case error:
		e = &Err{
			Status: http.StatusInternalServerError,
			Cause:  v,
			Reason: v.Error(),
		}
	case string:
		e = &Err{
			Message: v,
			Status:  http.StatusInternalServerError,
			Cause:   errors.New(v),
		}
	default:
		e = &Err{
			Reason: fmt.Sprint(err),
			Status: http.StatusInternalServerError,
		}
	}
	if e.Message == "" {
		e.Message = http.StatusText(e.Status)
	}

	return e
}

func throwErrorStatus(err error, s int, f string, args ...interface{}) {
	if err == nil {
		return
	}
	e := FromError(err)
	if e.Message == "" || e.Message == http.StatusText(e.Status) {
		e.Message = fmt.Sprintf(f, args...)
	}
	if s != http.StatusInternalServerError {
		e.Status = s
	}
	if e.Reason == "" {
		e.Reason = err.Error()
	}
	panic(e)
}

func throwError(err error, fmt string, args ...interface{}) {
	if err == nil {
		return
	}
	throwErrorStatus(err, detectStatusCode(err), fmt, args...)
}

func swallowError(err error, fmt string, args ...interface{}) {
	if err == nil {
		return
	}
	zap.S().With("error", err).Warnf(fmt, args...)
}

func detectStatusCode(err error) int {
	if err == gorm.ErrRecordNotFound {
		return http.StatusNotFound
	}
	return http.StatusInternalServerError
}
