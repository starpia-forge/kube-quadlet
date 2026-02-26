package converter

import (
	"strings"
	"unicode"
)

// SplitArgs splits a string into arguments, respecting quotes.
// This is a simplified implementation of shlex.Split.
func SplitArgs(s string) ([]string, error) {
	var args []string
	var currentArg strings.Builder
	inQuote := false
	quoteChar := rune(0)
	escaped := false

	for _, r := range s {
		if escaped {
			currentArg.WriteRune(r)
			escaped = false
			continue
		}

		if r == '\\' {
			escaped = true
			continue
		}

		if inQuote {
			if r == quoteChar {
				inQuote = false
				quoteChar = 0
			} else {
				currentArg.WriteRune(r)
			}
		} else {
			if r == '"' || r == '\'' {
				inQuote = true
				quoteChar = r
			} else if unicode.IsSpace(r) {
				if currentArg.Len() > 0 {
					args = append(args, currentArg.String())
					currentArg.Reset()
				}
			} else {
				currentArg.WriteRune(r)
			}
		}
	}

	if currentArg.Len() > 0 {
		args = append(args, currentArg.String())
	}

	return args, nil
}
