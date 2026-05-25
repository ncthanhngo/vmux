package rpc

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
)

// Handler processes a single request. params is the raw JSON of the request's
// "params" field (may be empty). The returned value is JSON-marshalled into the
// response result; returning an *Error produces a JSON-RPC error reply.
type Handler func(ctx context.Context, params json.RawMessage) (any, error)

// router maps method names to handlers. Registration happens at startup
// (single-threaded), dispatch is concurrent and read-only.
type router struct {
	mu       sync.RWMutex
	handlers map[string]Handler
}

func newRouter() *router {
	return &router{handlers: make(map[string]Handler)}
}

func (r *router) register(method string, h Handler) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, dup := r.handlers[method]; dup {
		panic(fmt.Sprintf("rpc: method %q already registered", method))
	}
	r.handlers[method] = h
}

func (r *router) lookup(method string) (Handler, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	h, ok := r.handlers[method]
	return h, ok
}
