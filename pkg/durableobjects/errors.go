package durableobjects

import (
	"errors"
	"fmt"
)

type ErrorCode string

const (
	CodeBadRequest       ErrorCode = "bad_request"
	CodeUnknownNamespace ErrorCode = "unknown_namespace"
	CodeActorStartFailed ErrorCode = "actor_start_failed"
	CodeMethodNotFound   ErrorCode = "method_not_found"
	CodeStorageError     ErrorCode = "storage_error"
	CodeExecutionError   ErrorCode = "execution_error"
	CodeTimeout          ErrorCode = "timeout"
)

type Error struct {
	Code    ErrorCode
	Message string
	Err     error
}

func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	if e.Message != "" {
		return e.Message
	}
	if e.Err != nil {
		return e.Err.Error()
	}
	return string(e.Code)
}

func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

func coded(code ErrorCode, format string, args ...any) *Error {
	return &Error{Code: code, Message: fmt.Sprintf(format, args...)}
}

func wrap(code ErrorCode, message string, err error) *Error {
	if err == nil {
		return &Error{Code: code, Message: message}
	}
	return &Error{Code: code, Message: message + ": " + err.Error(), Err: err}
}

func CodeOf(err error) ErrorCode {
	var e *Error
	if errors.As(err, &e) && e.Code != "" {
		return e.Code
	}
	return CodeExecutionError
}
