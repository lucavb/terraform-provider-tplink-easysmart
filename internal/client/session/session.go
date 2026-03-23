package session

import "regexp"

type LoginStatus int

const (
	LoginStatusUnknown LoginStatus = iota
	LoginStatusSuccess
	LoginStatusFailure
)

var logonInfoPattern = regexp.MustCompile(`var\s+logonInfo\s*=\s*new\s+Array\(\s*([0-9-]+)`)

func ClassifyLoginResponse(body string) (LoginStatus, int) {
	matches := logonInfoPattern.FindStringSubmatch(body)
	if len(matches) != 2 {
		return LoginStatusUnknown, -1
	}

	switch matches[1] {
	case "0":
		return LoginStatusSuccess, 0
	default:
		return LoginStatusFailure, 1
	}
}

func IsLoginPage(body string) bool {
	if status, _ := ClassifyLoginResponse(body); status != LoginStatusUnknown {
		return true
	}

	return regexp.MustCompile(`action="/logon\.cgi"|action=/logon\.cgi|name="password"|name=password`).MatchString(body)
}
