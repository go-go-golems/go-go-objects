package durableobjects

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"strings"
)

const minimumBindingKeyBytes = 32

// Dispatcher is the minimal execution contract used by BoundDispatcher.
type Dispatcher interface {
	Dispatch(ctx context.Context, env Envelope) (Result, error)
}

var _ Dispatcher = (*Manager)(nil)

// BoundDispatcher derives opaque physical object identifiers from a trusted
// actor identifier and an allowlisted namespace. Callers cannot supply an
// object name, preventing an authenticated user from selecting another user's
// object through the low-level manager API.
type BoundDispatcher struct {
	dispatcher Dispatcher
	key        []byte
	namespaces map[string]struct{}
}

// NewBoundDispatcher constructs an actor-bound dispatch boundary. The binding
// key must contain at least 256 bits and should be generated randomly and kept
// with the application's persistent secrets. At least one namespace must be
// explicitly allowlisted.
func NewBoundDispatcher(dispatcher Dispatcher, bindingKey []byte, namespaces []string) (*BoundDispatcher, error) {
	if dispatcher == nil {
		return nil, coded(CodeBadRequest, "bound dispatcher requires a dispatcher")
	}
	if len(bindingKey) < minimumBindingKeyBytes {
		return nil, coded(CodeBadRequest, "bound dispatcher binding key must contain at least %d bytes", minimumBindingKeyBytes)
	}
	allowed := make(map[string]struct{}, len(namespaces))
	for _, namespace := range namespaces {
		namespace = strings.TrimSpace(namespace)
		if err := validateNamespace(namespace); err != nil {
			return nil, err
		}
		allowed[namespace] = struct{}{}
	}
	if len(allowed) == 0 {
		return nil, coded(CodeBadRequest, "bound dispatcher requires at least one allowed namespace")
	}
	return &BoundDispatcher{
		dispatcher: dispatcher,
		key:        append([]byte(nil), bindingKey...),
		namespaces: allowed,
	}, nil
}

// ObjectIDForActor derives the physical object identifier for actorID in an
// allowed namespace. The returned name contains no actor identifier.
func (b *BoundDispatcher) ObjectIDForActor(namespace, actorID string) (ObjectID, error) {
	if b == nil || b.dispatcher == nil {
		return ObjectID{}, coded(CodeExecutionError, "bound dispatcher is not initialized")
	}
	namespace = strings.TrimSpace(namespace)
	if _, ok := b.namespaces[namespace]; !ok {
		return ObjectID{}, coded(CodeUnknownNamespace, "namespace is not available through actor-bound dispatch")
	}
	actorID = strings.TrimSpace(actorID)
	if actorID == "" {
		return ObjectID{}, coded(CodeBadRequest, "actor-bound dispatch requires an actor identifier")
	}
	mac := hmac.New(sha256.New, b.key)
	_, _ = mac.Write([]byte(namespace))
	_, _ = mac.Write([]byte{0})
	_, _ = mac.Write([]byte(actorID))
	name := "v1_" + base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return NewObjectID(namespace, name)
}

// DispatchForActor derives and installs the physical object ID before calling
// the underlying dispatcher. env.ID must be empty so accidental or malicious
// object-name injection is rejected rather than silently ignored.
func (b *BoundDispatcher) DispatchForActor(ctx context.Context, actorID, namespace string, env Envelope) (Result, error) {
	if env.ID.Namespace != "" || env.ID.Name != "" || env.ID.Hash != "" {
		return Result{}, coded(CodeBadRequest, "actor-bound dispatch does not accept a caller-supplied object id")
	}
	id, err := b.ObjectIDForActor(namespace, actorID)
	if err != nil {
		return Result{}, err
	}
	env.ID = id
	return b.dispatcher.Dispatch(ctx, env)
}

// RPCForActor dispatches one RPC operation to the actor's private object.
func (b *BoundDispatcher) RPCForActor(ctx context.Context, actorID, namespace, method string, args []byte) (Result, error) {
	return b.DispatchForActor(ctx, actorID, namespace, Envelope{Kind: KindRPC, Method: method, ArgsJSON: append([]byte(nil), args...)})
}

// FetchForActor dispatches one fetch operation to the actor's private object.
func (b *BoundDispatcher) FetchForActor(ctx context.Context, actorID, namespace string, request *FetchRequest) (Result, error) {
	return b.DispatchForActor(ctx, actorID, namespace, Envelope{Kind: KindFetch, Request: request})
}
