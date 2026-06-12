package durableobjects

import (
	"encoding/json"
	"time"
)

type Kind string

const (
	KindRPC   Kind = "rpc"
	KindFetch Kind = "fetch"
	KindAlarm Kind = "alarm"
)

type Envelope struct {
	Kind      Kind
	ID        ObjectID
	Method    string
	ArgsJSON  json.RawMessage
	Request   *FetchRequest
	Deadline  time.Time
	RequestID string
}

type Result struct {
	ValueJSON json.RawMessage
	Response  *FetchResponse
}

type FetchRequest struct {
	Method  string            `json:"method"`
	URL     string            `json:"url"`
	Path    string            `json:"path"`
	Query   map[string]any    `json:"query,omitempty"`
	Headers map[string]string `json:"headers,omitempty"`
	Body    any               `json:"body,omitempty"`
	RawBody string            `json:"rawBody,omitempty"`
}

type FetchResponse struct {
	Status  int               `json:"status"`
	Headers map[string]string `json:"headers,omitempty"`
	Body    any               `json:"body,omitempty"`
}
