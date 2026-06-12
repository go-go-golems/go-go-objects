package durableobjects

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
)

type GatewayOptions struct {
	MaxRequestBytes int64
	DevErrors       bool
}

type Gateway struct {
	manager *Manager
	opts    GatewayOptions
}

func NewGateway(manager *Manager, opts GatewayOptions) *Gateway {
	if opts.MaxRequestBytes <= 0 {
		opts.MaxRequestBytes = 64 << 20
	}
	return &Gateway{manager: manager, opts: opts}
}

func (g *Gateway) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch {
	case strings.HasPrefix(r.URL.Path, "/rpc/"):
		g.serveRPC(w, r)
	case strings.HasPrefix(r.URL.Path, "/fetch/"):
		g.serveFetch(w, r)
	default:
		http.NotFound(w, r)
	}
}

func (g *Gateway) serveRPC(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, coded(CodeBadRequest, "rpc endpoint requires POST"), g.opts.DevErrors)
		return
	}
	namespace, name, method, err := parseRPCPath(r.URL.Path)
	if err != nil {
		writeError(w, err, g.opts.DevErrors)
		return
	}
	id, err := NewObjectID(namespace, name)
	if err != nil {
		writeError(w, err, g.opts.DevErrors)
		return
	}
	body, err := readBody(w, r, g.opts.MaxRequestBytes)
	if err != nil {
		writeError(w, err, g.opts.DevErrors)
		return
	}
	result, err := g.manager.Dispatch(r.Context(), Envelope{Kind: KindRPC, ID: id, Method: method, ArgsJSON: body})
	if err != nil {
		writeError(w, err, g.opts.DevErrors)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"ok":true,"result":`))
	if len(result.ValueJSON) == 0 {
		_, _ = w.Write([]byte(`null`))
	} else {
		_, _ = w.Write(result.ValueJSON)
	}
	_, _ = w.Write([]byte("}\n"))
}

func (g *Gateway) serveFetch(w http.ResponseWriter, r *http.Request) {
	namespace, name, rest, err := parseFetchPath(r.URL.Path)
	if err != nil {
		writeError(w, err, g.opts.DevErrors)
		return
	}
	id, err := NewObjectID(namespace, name)
	if err != nil {
		writeError(w, err, g.opts.DevErrors)
		return
	}
	body, raw, err := parseGatewayBody(w, r, g.opts.MaxRequestBytes)
	if err != nil {
		writeError(w, err, g.opts.DevErrors)
		return
	}
	result, err := g.manager.Dispatch(r.Context(), Envelope{Kind: KindFetch, ID: id, Request: newFetchRequest(r, rest, body, raw)})
	if err != nil {
		writeError(w, err, g.opts.DevErrors)
		return
	}
	writeFetch(w, result.Response)
}

func parseRPCPath(path string) (namespace, name, method string, err error) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 4 || parts[0] != "rpc" {
		return "", "", "", coded(CodeBadRequest, "rpc path must be /rpc/:namespace/:name/:method")
	}
	return parts[1], parts[2], parts[3], nil
}

func parseFetchPath(path string) (namespace, name, rest string, err error) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) < 3 || parts[0] != "fetch" {
		return "", "", "", coded(CodeBadRequest, "fetch path must be /fetch/:namespace/:name/*")
	}
	if len(parts) > 3 {
		rest = "/" + strings.Join(parts[3:], "/")
	} else {
		rest = "/"
	}
	return parts[1], parts[2], rest, nil
}

func readBody(w http.ResponseWriter, r *http.Request, maxBytes int64) ([]byte, error) {
	if r.Body == nil {
		return nil, nil
	}
	data, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxBytes))
	if err != nil {
		return nil, wrap(CodeBadRequest, "read request body", err)
	}
	return data, nil
}

func parseGatewayBody(w http.ResponseWriter, r *http.Request, maxBytes int64) (any, string, error) {
	data, err := readBody(w, r, maxBytes)
	if err != nil {
		return nil, "", err
	}
	raw := string(data)
	if len(data) == 0 {
		return nil, raw, nil
	}
	if strings.Contains(strings.ToLower(r.Header.Get("Content-Type")), "application/json") {
		var v any
		if err := json.Unmarshal(data, &v); err != nil {
			return nil, raw, wrap(CodeBadRequest, "decode JSON body", err)
		}
		return v, raw, nil
	}
	return raw, raw, nil
}

func newFetchRequest(r *http.Request, rest string, body any, raw string) *FetchRequest {
	query := map[string]any{}
	for k, vals := range r.URL.Query() {
		if len(vals) == 1 {
			query[k] = vals[0]
		} else {
			query[k] = vals
		}
	}
	headers := map[string]string{}
	for k, vals := range r.Header {
		headers[k] = strings.Join(vals, ", ")
	}
	return &FetchRequest{Method: r.Method, URL: r.URL.String(), Path: rest, Query: query, Headers: headers, Body: body, RawBody: raw}
}

func writeFetch(w http.ResponseWriter, response *FetchResponse) {
	if response == nil {
		response = &FetchResponse{Status: http.StatusNoContent}
	}
	if response.Status == 0 {
		response.Status = http.StatusOK
	}
	for k, v := range response.Headers {
		w.Header().Set(k, v)
	}
	if response.Body == nil {
		w.WriteHeader(response.Status)
		return
	}
	if _, ok := response.Body.(string); ok && w.Header().Get("Content-Type") == "" {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(response.Status)
		_, _ = w.Write([]byte(response.Body.(string)))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(response.Status)
	_ = json.NewEncoder(w).Encode(response.Body)
}

func writeError(w http.ResponseWriter, err error, dev bool) {
	code := CodeOf(err)
	status := http.StatusInternalServerError
	switch code {
	case CodeBadRequest:
		status = http.StatusBadRequest
	case CodeUnknownNamespace, CodeMethodNotFound:
		status = http.StatusNotFound
	case CodeTimeout:
		status = http.StatusGatewayTimeout
	}
	message := "internal server error"
	if dev || status < 500 {
		message = err.Error()
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{"ok": false, "error": map[string]any{"code": code, "message": message}})
}
