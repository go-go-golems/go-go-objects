package durableobjects

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
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
	sum := sha256.Sum256([]byte(namespace + "\x00" + name))
	return ObjectID{Namespace: namespace, Name: name, Hash: hex.EncodeToString(sum[:])}, nil
}

func (id ObjectID) IsZero() bool {
	return id.Namespace == "" || id.Name == "" || id.Hash == ""
}
