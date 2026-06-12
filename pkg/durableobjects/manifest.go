package durableobjects

import "strings"

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
