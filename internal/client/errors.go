package client

import "errors"

var (
	ErrLoginFailed          = errors.New("login failed")
	ErrSessionInvalid       = errors.New("session invalid")
	ErrPageReturnedLogin    = errors.New("page returned login html")
	ErrUnsupportedPageModel = errors.New("unsupported page model")
)
