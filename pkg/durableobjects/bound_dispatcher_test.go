package durableobjects

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

type recordingDispatcher struct {
	envelopes []Envelope
}

func (d *recordingDispatcher) Dispatch(_ context.Context, env Envelope) (Result, error) {
	d.envelopes = append(d.envelopes, env)
	return Result{ValueJSON: json.RawMessage(`true`)}, nil
}

func TestBoundDispatcherDerivesStableIsolatedObjectIDs(t *testing.T) {
	t.Parallel()
	recorder := &recordingDispatcher{}
	dispatcher, err := NewBoundDispatcher(recorder, []byte(strings.Repeat("k", minimumBindingKeyBytes)), []string{"PROFILE"})
	if err != nil {
		t.Fatalf("NewBoundDispatcher: %v", err)
	}
	aliceFirst, err := dispatcher.ObjectIDForActor("PROFILE", "issuer\x00alice")
	if err != nil {
		t.Fatalf("alice id: %v", err)
	}
	aliceSecond, err := dispatcher.ObjectIDForActor("PROFILE", "issuer\x00alice")
	if err != nil {
		t.Fatalf("alice second id: %v", err)
	}
	bob, err := dispatcher.ObjectIDForActor("PROFILE", "issuer\x00bob")
	if err != nil {
		t.Fatalf("bob id: %v", err)
	}
	if aliceFirst != aliceSecond {
		t.Fatalf("derivation is not stable: %#v != %#v", aliceFirst, aliceSecond)
	}
	if aliceFirst == bob || aliceFirst.Name == bob.Name {
		t.Fatalf("two actors share object identity: alice=%#v bob=%#v", aliceFirst, bob)
	}
	if strings.Contains(aliceFirst.Name, "alice") || strings.Contains(bob.Name, "bob") {
		t.Fatalf("derived names reveal actor identifiers: alice=%q bob=%q", aliceFirst.Name, bob.Name)
	}
}

func TestBoundDispatcherRejectsNamespaceAndObjectInjection(t *testing.T) {
	t.Parallel()
	recorder := &recordingDispatcher{}
	dispatcher, err := NewBoundDispatcher(recorder, []byte(strings.Repeat("k", minimumBindingKeyBytes)), []string{"PROFILE"})
	if err != nil {
		t.Fatalf("NewBoundDispatcher: %v", err)
	}
	if _, err := dispatcher.RPCForActor(context.Background(), "alice", "ADMIN", "read", nil); CodeOf(err) != CodeUnknownNamespace {
		t.Fatalf("namespace error=%v code=%s", err, CodeOf(err))
	}
	injectedID, err := NewObjectID("PROFILE", "victim")
	if err != nil {
		t.Fatal(err)
	}
	_, err = dispatcher.DispatchForActor(context.Background(), "alice", "PROFILE", Envelope{Kind: KindRPC, ID: injectedID, Method: "read"})
	if CodeOf(err) != CodeBadRequest {
		t.Fatalf("injection error=%v code=%s", err, CodeOf(err))
	}
	if len(recorder.envelopes) != 0 {
		t.Fatalf("underlying dispatcher received rejected envelopes: %#v", recorder.envelopes)
	}
}

func TestNewBoundDispatcherValidatesSecurityInputs(t *testing.T) {
	t.Parallel()
	recorder := &recordingDispatcher{}
	tests := []struct {
		name       string
		dispatcher Dispatcher
		key        []byte
		namespaces []string
	}{
		{name: "nil dispatcher", key: []byte(strings.Repeat("k", minimumBindingKeyBytes)), namespaces: []string{"PROFILE"}},
		{name: "short key", dispatcher: recorder, key: []byte("short"), namespaces: []string{"PROFILE"}},
		{name: "no namespaces", dispatcher: recorder, key: []byte(strings.Repeat("k", minimumBindingKeyBytes))},
		{name: "invalid namespace", dispatcher: recorder, key: []byte(strings.Repeat("k", minimumBindingKeyBytes)), namespaces: []string{"../PROFILE"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := NewBoundDispatcher(test.dispatcher, test.key, test.namespaces); err == nil {
				t.Fatal("NewBoundDispatcher succeeded, want error")
			}
		})
	}
}

func TestBoundDispatcherCopiesBindingKey(t *testing.T) {
	t.Parallel()
	key := []byte(strings.Repeat("k", minimumBindingKeyBytes))
	dispatcher, err := NewBoundDispatcher(&recordingDispatcher{}, key, []string{"PROFILE"})
	if err != nil {
		t.Fatal(err)
	}
	before, err := dispatcher.ObjectIDForActor("PROFILE", "alice")
	if err != nil {
		t.Fatal(err)
	}
	for i := range key {
		key[i] = 'x'
	}
	after, err := dispatcher.ObjectIDForActor("PROFILE", "alice")
	if err != nil {
		t.Fatal(err)
	}
	if before != after {
		t.Fatalf("caller mutation changed key: before=%#v after=%#v", before, after)
	}
}

func TestBoundDispatcherEnforcesTwoActorStorageIsolation(t *testing.T) {
	manager := newTestManager(t)
	dispatcher, err := NewBoundDispatcher(manager, []byte(strings.Repeat("k", minimumBindingKeyBytes)), []string{"COUNTER"})
	if err != nil {
		t.Fatal(err)
	}
	encodeArgs := func(values ...any) []byte {
		data, marshalErr := json.Marshal(values)
		if marshalErr != nil {
			t.Fatal(marshalErr)
		}
		return data
	}
	alice, err := dispatcher.RPCForActor(context.Background(), "issuer\x00alice", "COUNTER", "increment", encodeArgs(7))
	if err != nil {
		t.Fatalf("alice increment: %v", err)
	}
	bob, err := dispatcher.RPCForActor(context.Background(), "issuer\x00bob", "COUNTER", "value", encodeArgs())
	if err != nil {
		t.Fatalf("bob read: %v", err)
	}
	if string(alice.ValueJSON) != "7" {
		t.Fatalf("alice value=%s, want 7", alice.ValueJSON)
	}
	if string(bob.ValueJSON) != "0" {
		t.Fatalf("bob observed alice state: value=%s, want 0", bob.ValueJSON)
	}
}
