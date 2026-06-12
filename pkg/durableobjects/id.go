package durableobjects

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"unicode"
)

type ObjectID struct {
	Namespace string `json:"namespace"`
	Name      string `json:"name"`
	Hash      string `json:"hash"`
}

func NewObjectID(namespace, name string) (ObjectID, error) {
	namespace = strings.TrimSpace(namespace)
	name = strings.TrimSpace(name)
	if namespace == "" {
		return ObjectID{}, coded(CodeBadRequest, "object namespace is required")
	}
	if name == "" {
		return ObjectID{}, coded(CodeBadRequest, "object name is required")
	}
	if err := validateNamespace(namespace); err != nil {
		return ObjectID{}, err
	}
	if err := validateObjectName(name); err != nil {
		return ObjectID{}, err
	}
	sum := sha256.Sum256([]byte(namespace + "\x00" + name))
	return ObjectID{Namespace: namespace, Name: name, Hash: hex.EncodeToString(sum[:])}, nil
}

func validateNamespace(namespace string) error {
	if namespace == "." || namespace == ".." || strings.ContainsAny(namespace, `/\\`) {
		return coded(CodeBadRequest, "object namespace %q is not a safe namespace segment", namespace)
	}
	if len(namespace) > 128 {
		return coded(CodeBadRequest, "object namespace is too long")
	}
	for _, r := range namespace {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '-' || r == '.' {
			continue
		}
		return coded(CodeBadRequest, "object namespace %q contains invalid character %q", namespace, r)
	}
	return nil
}

func validateObjectName(name string) error {
	if name == "." || name == ".." || strings.ContainsAny(name, `/\\`) {
		return coded(CodeBadRequest, "object name %q is not a safe object-name segment", name)
	}
	if len(name) > 1024 {
		return coded(CodeBadRequest, "object name is too long")
	}
	for _, r := range name {
		if unicode.IsControl(r) {
			return coded(CodeBadRequest, "object name contains a control character")
		}
	}
	return nil
}

func (id ObjectID) IsZero() bool {
	return id.Namespace == "" || id.Name == "" || id.Hash == ""
}
