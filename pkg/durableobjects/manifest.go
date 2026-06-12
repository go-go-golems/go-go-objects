package durableobjects

import (
	"strings"
	"unicode"
)

type Manifest struct {
	Objects map[string]string `json:"objects" yaml:"objects"`
}

func (m Manifest) ClassForNamespace(namespace string) (string, bool) {
	if m.Objects == nil {
		return "", false
	}
	className, ok := m.Objects[strings.TrimSpace(namespace)]
	className = strings.TrimSpace(className)
	return className, ok && className != ""
}

func (m Manifest) IsZero() bool {
	return len(m.Objects) == 0
}

func (m Manifest) Validate() error {
	if len(m.Objects) == 0 {
		return coded(CodeBadRequest, "durable object manifest must define at least one object namespace")
	}
	for namespace, className := range m.Objects {
		if strings.TrimSpace(namespace) == "" {
			return coded(CodeBadRequest, "durable object manifest contains an empty namespace")
		}
		if strings.TrimSpace(className) == "" {
			return coded(CodeBadRequest, "durable object namespace %q has an empty class name", namespace)
		}
	}
	return nil
}

func ExportNameToNamespace(exportName string) string {
	exportName = strings.TrimSpace(exportName)
	if exportName == "" {
		return ""
	}
	runes := []rune(exportName)
	var out []rune
	lastUnderscore := false
	for i, r := range runes {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			if unicode.IsUpper(r) && i > 0 && len(out) > 0 && !lastUnderscore {
				prev := runes[i-1]
				var next rune
				if i+1 < len(runes) {
					next = runes[i+1]
				}
				if unicode.IsLower(prev) || unicode.IsDigit(prev) || (unicode.IsUpper(prev) && next != 0 && unicode.IsLower(next)) {
					out = append(out, '_')
				}
			}
			out = append(out, unicode.ToUpper(r))
			lastUnderscore = false
			continue
		}
		if len(out) > 0 && !lastUnderscore {
			out = append(out, '_')
			lastUnderscore = true
		}
	}
	for len(out) > 0 && out[len(out)-1] == '_' {
		out = out[:len(out)-1]
	}
	return string(out)
}
