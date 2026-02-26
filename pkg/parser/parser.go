package parser

import (
	"bufio"
	"io"
	"strings"
)

type Option struct {
	Key   string
	Value string
}

type Unit struct {
	Sections map[string][]Option
}

func Parse(r io.Reader) (*Unit, error) {
	scanner := bufio.NewScanner(r)
	unit := &Unit{
		Sections: make(map[string][]Option),
	}

	var currentSection string
	var buffer strings.Builder
	inContinuation := false

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Ignore empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}

		if inContinuation {
			buffer.WriteString(" ")
		} else {
			buffer.Reset()
		}

		isContinuation := strings.HasSuffix(line, "\\")
		content := line
		if isContinuation {
			content = line[:len(line)-1]
			// The backslash might be preceded by space, which is part of value?
            // Usually spaces before backslash are discarded in systemd?
            // "The backslash and the newline character that follows it are replaced by a space character."
            // So spaces before backslash are preserved?
            // "Trailing whitespace is removed." - systemd.unit(5)
            // So `Val \` -> `Val` (space preserved?) No, `TrimSpace` removes trailing space.
            // If input is `Val \ `, `TrimSpace` -> `Val \`. `HasSuffix` is true. `content` -> `Val `.
            // This seems correct.
		}

		buffer.WriteString(content)

		if isContinuation {
			inContinuation = true
			continue
		}

		// Process complete line
		fullLine := buffer.String()
		inContinuation = false

		if strings.HasPrefix(fullLine, "[") && strings.HasSuffix(fullLine, "]") {
			currentSection = fullLine[1 : len(fullLine)-1]
		} else {
			if currentSection == "" {
				continue
			}
			parts := strings.SplitN(fullLine, "=", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				value := strings.TrimSpace(parts[1])

				if _, ok := unit.Sections[currentSection]; !ok {
					unit.Sections[currentSection] = []Option{}
				}
				unit.Sections[currentSection] = append(unit.Sections[currentSection], Option{
					Key:   key,
					Value: value,
				})
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return unit, nil
}
