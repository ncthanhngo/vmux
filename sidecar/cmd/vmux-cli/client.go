package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"sync"

	"github.com/vmux/sidecar/internal/rpc"
)

// client is a minimal JSON-RPC client over the vmux Unix socket. It matches
// responses to requests by id and forwards notifications to onNotify.
type client struct {
	conn     net.Conn
	onNotify func(method string, params json.RawMessage)

	mu      sync.Mutex
	nextID  int
	pending map[int]chan rpc.Response
}

func dial(socketPath string) (*client, error) {
	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		return nil, err
	}
	c := &client{conn: conn, pending: make(map[int]chan rpc.Response)}
	go c.readLoop()
	return c, nil
}

func (c *client) readLoop() {
	r := bufio.NewReaderSize(c.conn, 64<<10)
	for {
		line, err := r.ReadBytes('\n')
		if err != nil {
			c.failPending(err)
			return
		}
		var probe struct {
			ID     *json.RawMessage `json:"id"`
			Method string           `json:"method"`
			Params json.RawMessage  `json:"params"`
		}
		if json.Unmarshal(line, &probe) != nil {
			continue
		}
		if probe.ID == nil && probe.Method != "" {
			if c.onNotify != nil {
				c.onNotify(probe.Method, probe.Params)
			}
			continue
		}
		var resp rpc.Response
		if json.Unmarshal(line, &resp) != nil || resp.ID == nil {
			continue
		}
		var idNum int
		if json.Unmarshal(*resp.ID, &idNum) != nil {
			continue
		}
		c.deliver(idNum, resp)
	}
}

func (c *client) deliver(id int, resp rpc.Response) {
	c.mu.Lock()
	ch := c.pending[id]
	delete(c.pending, id)
	c.mu.Unlock()
	if ch != nil {
		ch <- resp
	}
}

func (c *client) failPending(err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for id, ch := range c.pending {
		close(ch)
		delete(c.pending, id)
	}
}

// call sends a request and waits for its response, unmarshalling result into out.
func (c *client) call(method string, params any, out any) error {
	paramBytes, err := json.Marshal(params)
	if err != nil {
		return err
	}
	c.mu.Lock()
	c.nextID++
	id := c.nextID
	ch := make(chan rpc.Response, 1)
	c.pending[id] = ch
	c.mu.Unlock()

	idRaw := json.RawMessage(fmt.Sprintf("%d", id))
	prm := json.RawMessage(paramBytes)
	req := rpc.Request{JSONRPC: "2.0", ID: &idRaw, Method: method, Params: prm}
	reqBytes, _ := json.Marshal(req)
	if _, err := c.conn.Write(append(reqBytes, '\n')); err != nil {
		return err
	}

	resp, ok := <-ch
	if !ok {
		return fmt.Errorf("connection closed before response to %s", method)
	}
	if resp.Error != nil {
		return fmt.Errorf("%s: %s", method, resp.Error.Message)
	}
	if out != nil && len(resp.Result) > 0 {
		return json.Unmarshal(resp.Result, out)
	}
	return nil
}

func (c *client) close() { c.conn.Close() }
