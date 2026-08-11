package client

import "errors"

var (
	ErrLoginFailed          = errors.New("login failed")
	ErrSessionInvalid       = errors.New("session invalid")
	ErrPageReturnedLogin    = errors.New("switch returned login page: session expired or lost; re-run apply")
	ErrUnsupportedPageModel = errors.New("unsupported page model")
)
