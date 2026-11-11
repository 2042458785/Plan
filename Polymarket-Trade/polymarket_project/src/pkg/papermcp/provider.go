package papermcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

var ErrUnavailable = errors.New("MCP unavailable")
var ErrOldSession = errors.New("obsolete MCP session")

type Status struct {
	Ready   bool
	Session uint64
	Error   string
}

type Provider struct {
	path string
	args []string

	mu      sync.RWMutex
	client  *mcp.ClientSession
	status  Status
	restart chan struct{}
}

func New(path string, args ...string) *Provider {
	return &Provider{path: path, args: args, restart: make(chan struct{}, 1)}
}

func (p *Provider) Status() Status {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.status
}

func (p *Provider) Reconnect() {
	p.mu.Lock()
	p.status.Ready = false
	p.mu.Unlock()
	select {
	case p.restart <- struct{}{}:
	default:
	}
}

func (p *Provider) Run(ctx context.Context) {
	backoff := time.Second
	for ctx.Err() == nil {
		started := time.Now()
		manual := p.serve(ctx)
		if ctx.Err() != nil {
			return
		}
		if manual || time.Since(started) >= 16*time.Second {
			backoff = time.Second
		}
		if manual {
			continue
		}
		timer := time.NewTimer(backoff)
		select {
		case <-ctx.Done():
		case <-p.restart:
			backoff = time.Second
		case <-timer.C:
			backoff = min(backoff*2, 16*time.Second)
		}
		timer.Stop()
	}
}

func (p *Provider) serve(ctx context.Context) bool {
	childCtx, stopChild := context.WithCancel(ctx)
	defer stopChild()
	tail := &stderrTail{}
	command := exec.CommandContext(childCtx, p.path, p.args...)
	command.Stderr = tail
	transport := &mcp.CommandTransport{Command: command, TerminateDuration: 300 * time.Millisecond}
	client := mcp.NewClient(&mcp.Implementation{Name: "polymarket-taker-sim", Version: "0.1.0"}, nil)
	handshake, cancel := context.WithTimeout(ctx, 5*time.Second)
	session, err := client.Connect(handshake, transport, nil)
	if err == nil {
		err = verifyTools(handshake, session)
	}
	cancel()
	if err != nil {
		p.down(err, tail)
		stopChild()
		if session != nil {
			_ = session.Close()
		}
		return false
	}

	p.mu.Lock()
	p.client = session
	p.status.Session++
	p.status.Ready = true
	p.status.Error = ""
	p.mu.Unlock()

	closed := make(chan error, 1)
	go func() { closed <- session.Wait() }()
	manual := false
	select {
	case <-ctx.Done():
	case <-p.restart:
		manual = true
	case err = <-closed:
		if err == nil {
			err = errors.New("MCP subprocess exited")
		}
	}
	p.down(err, tail)
	stopChild()
	_ = session.Close()
	return manual
}

func verifyTools(ctx context.Context, session *mcp.ClientSession) error {
	result, err := session.ListTools(ctx, nil)
	if err != nil {
		return err
	}
	names := map[string]bool{}
	for _, tool := range result.Tools {
		names[tool.Name] = true
	}
	if !names["get_pair_snapshot"] || !names["simulate_fill"] {
		return errors.New("missing paper adapter tools")
	}
	return nil
}

func (p *Provider) down(err error, tail *stderrTail) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.client = nil
	p.status.Ready = false
	p.status.Error = ""
	if err != nil {
		p.status.Error = err.Error()
		if detail := tail.last(); detail != "" {
			p.status.Error += " | " + detail
		}
	}
}

func (p *Provider) Snapshot(ctx context.Context) (Snapshot, error) {
	var result Snapshot
	started := time.Now()
	session, err := p.call(ctx, 0, "get_pair_snapshot", map[string]any{}, &result)
	result.Session = session
	result.RequestedAt = started
	return result, err
}

func (p *Provider) Fill(ctx context.Context, request FillRequest) (FillResult, error) {
	var result FillResult
	_, err := p.call(ctx, request.Session, "simulate_fill", request, &result)
	return result, err
}

func (p *Provider) call(ctx context.Context, wantSession uint64, name string, args any, output any) (uint64, error) {
	p.mu.RLock()
	client, status := p.client, p.status
	p.mu.RUnlock()
	if client == nil || !status.Ready {
		return 0, ErrUnavailable
	}
	if wantSession != 0 && wantSession != status.Session {
		return 0, ErrOldSession
	}
	result, err := client.CallTool(ctx, &mcp.CallToolParams{Name: name, Arguments: args})
	if err != nil {
		return 0, err
	}
	latest := p.Status()
	if !latest.Ready || latest.Session != status.Session {
		return 0, ErrOldSession
	}
	if result.IsError {
		for _, content := range result.Content {
			if value, ok := content.(*mcp.TextContent); ok {
				return 0, fmt.Errorf("%s: %.300s", name, value.Text)
			}
		}
		return 0, fmt.Errorf("%s failed", name)
	}
	if result.StructuredContent != nil {
		payload, err := json.Marshal(result.StructuredContent)
		if err != nil {
			return 0, err
		}
		return status.Session, json.Unmarshal(payload, output)
	}
	if len(result.Content) == 1 {
		if value, ok := result.Content[0].(*mcp.TextContent); ok {
			return status.Session, json.Unmarshal([]byte(value.Text), output)
		}
	}
	return 0, fmt.Errorf("%s returned no JSON payload", name)
}

type stderrTail struct {
	mu   sync.Mutex
	text string
}

func (t *stderrTail) Write(p []byte) (int, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	length := len(p)
	const limit = 640
	if len(p) > limit {
		p = p[len(p)-limit:]
		t.text = ""
	}
	t.text += string(p)
	if len(t.text) > limit {
		t.text = t.text[len(t.text)-limit:]
	}
	// exec expects the original write length even when retaining only the tail.
	return length, nil
}

func (t *stderrTail) last() string {
	t.mu.Lock()
	defer t.mu.Unlock()
	return strings.Join(strings.Fields(t.text), " ")
}
