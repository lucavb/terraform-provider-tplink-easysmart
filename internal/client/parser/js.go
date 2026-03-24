package parser

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"unicode"
)

var (
	errVariableNotFound = "variable %q not found"
	keyPattern          = regexp.MustCompile(`([{,]\s*)([A-Za-z_][A-Za-z0-9_]*)(\s*:)`)
	hexPattern          = regexp.MustCompile(`0x[0-9A-Fa-f]+`)
)

func ExtractObject(source string, name string) (map[string]any, error) {
	literal, err := extractAssignedLiteral(source, name)
	if err != nil {
		return nil, err
	}

	jsonLiteral, err := jsLiteralToJSON(literal)
	if err != nil {
		return nil, err
	}

	var out map[string]any
	if err := json.Unmarshal([]byte(jsonLiteral), &out); err != nil {
		return nil, fmt.Errorf("unmarshal object %q: %w", name, err)
	}
	return out, nil
}

func ExtractArray(source string, name string) ([]any, error) {
	literal, err := extractAssignedLiteral(source, name)
	if err != nil {
		return nil, err
	}

	jsonLiteral, err := jsLiteralToJSON(literal)
	if err != nil {
		return nil, err
	}

	var out []any
	if err := json.Unmarshal([]byte(jsonLiteral), &out); err != nil {
		return nil, fmt.Errorf("unmarshal array %q: %w", name, err)
	}
	return out, nil
}

func ExtractInt(source string, name string) (int, error) {
	literal, err := extractAssignedLiteral(source, name)
	if err != nil {
		return 0, err
	}

	value, err := strconv.Atoi(strings.TrimSpace(literal))
	if err != nil {
		return 0, fmt.Errorf("parse int %q: %w", name, err)
	}
	return value, nil
}

func extractAssignedLiteral(source string, name string) (string, error) {
	needle := "var " + name
	idx := strings.Index(source, needle)
	if idx == -1 {
		return "", fmt.Errorf(errVariableNotFound, name)
	}

	eqIdx := strings.Index(source[idx:], "=")
	if eqIdx == -1 {
		return "", fmt.Errorf("variable %q missing assignment", name)
	}

	start := idx + eqIdx + 1
	for start < len(source) && unicode.IsSpace(rune(source[start])) {
		start++
	}

	switch {
	case strings.HasPrefix(source[start:], "new Array("):
		openIdx := start + len("new Array")
		literal, next, err := extractBalanced(source, openIdx, '(', ')')
		if err != nil {
			return "", err
		}
		_ = next
		return "[" + strings.TrimSuffix(strings.TrimPrefix(literal, "("), ")") + "]", nil
	case strings.HasPrefix(source[start:], "{"):
		literal, _, err := extractBalanced(source, start, '{', '}')
		return literal, err
	case strings.HasPrefix(source[start:], "["):
		literal, _, err := extractBalanced(source, start, '[', ']')
		return literal, err
	default:
		end := start
		for end < len(source) && source[end] != ';' && source[end] != '\n' && source[end] != '\r' && source[end] != '<' {
			end++
		}
		return strings.TrimSpace(source[start:end]), nil
	}
}

func extractBalanced(source string, start int, open byte, close byte) (string, int, error) {
	if start >= len(source) || source[start] != open {
		return "", 0, fmt.Errorf("expected %q at %d", string(open), start)
	}

	depth := 0
	inString := false
	var stringDelimiter byte
	escaped := false

	for idx := start; idx < len(source); idx++ {
		ch := source[idx]

		if inString {
			if escaped {
				escaped = false
				continue
			}
			if ch == '\\' {
				escaped = true
				continue
			}
			if ch == stringDelimiter {
				inString = false
			}
			continue
		}

		if ch == '\'' || ch == '"' {
			inString = true
			stringDelimiter = ch
			continue
		}

		if ch == open {
			depth++
		}
		if ch == close {
			depth--
			if depth == 0 {
				return source[start : idx+1], idx + 1, nil
			}
		}
	}

	return "", 0, fmt.Errorf("unterminated %q literal", string(open))
}

func jsLiteralToJSON(literal string) (string, error) {
	normalized := strings.TrimSpace(literal)
	if normalized == "" {
		return "", fmt.Errorf("empty literal")
	}

	normalized = strings.ReplaceAll(normalized, `'`, `"`)
	normalized = keyPattern.ReplaceAllString(normalized, `${1}"${2}"${3}`)
	normalized = hexPattern.ReplaceAllStringFunc(normalized, func(value string) string {
		parsed, err := strconv.ParseInt(value[2:], 16, 64)
		if err != nil {
			return value
		}
		return strconv.FormatInt(parsed, 10)
	})

	return normalized, nil
}
