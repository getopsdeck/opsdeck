package mcp

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

// ---------------------------------------------------------------------------
// TestNewServer -- the constructor returns a valid, non-nil server.
// ---------------------------------------------------------------------------

func TestNewServer(t *testing.T) {
	srv := NewServer()
	if srv == nil {
		t.Fatal("NewServer returned nil")
	}
}

// ---------------------------------------------------------------------------
// TestServerHasTools -- after construction the server advertises 4 tools.
// We connect via in-memory transport and call tools/list.
// ---------------------------------------------------------------------------

func TestServerHasTools(t *testing.T) {
	srv := NewServer()

	client := sdkmcp.NewClient(&sdkmcp.Implementation{
		Name:    "test-client",
		Version: "0.0.1",
	}, nil)

	st, ct := sdkmcp.NewInMemoryTransports()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Run server in background.
	errCh := make(chan error, 1)
	go func() { errCh <- srv.Run(ctx, st) }()

	session, err := client.Connect(ctx, ct, nil)
	if err != nil {
		t.Fatalf("client.Connect failed: %v", err)
	}
	defer session.Close()

	res, err := session.ListTools(ctx, nil)
	if err != nil {
		t.Fatalf("ListTools failed: %v", err)
	}

	want := map[string]bool{
		"get_sessions":       false,
		"get_brief":          false,
		"get_costs":          false,
		"get_session_detail": false,
	}
	for _, tool := range res.Tools {
		if _, ok := want[tool.Name]; ok {
			want[tool.Name] = true
		}
	}
	for name, found := range want {
		if !found {
			t.Errorf("missing tool: %s", name)
		}
	}

	cancel()
	<-errCh
}

// ---------------------------------------------------------------------------
// TestGetSessionsTool -- calling get_sessions returns valid JSON.
// With no real sessions directory, we expect an empty array.
// ---------------------------------------------------------------------------

func TestGetSessionsTool(t *testing.T) {
	srv := NewServer()

	client := sdkmcp.NewClient(&sdkmcp.Implementation{
		Name:    "test-client",
		Version: "0.0.1",
	}, nil)

	st, ct := sdkmcp.NewInMemoryTransports()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	errCh := make(chan error, 1)
	go func() { errCh <- srv.Run(ctx, st) }()

	session, err := client.Connect(ctx, ct, nil)
	if err != nil {
		t.Fatalf("client.Connect failed: %v", err)
	}
	defer session.Close()

	res, err := session.CallTool(ctx, &sdkmcp.CallToolParams{
		Name: "get_sessions",
	})
	if err != nil {
		t.Fatalf("CallTool(get_sessions) failed: %v", err)
	}

	if res.IsError {
		t.Fatalf("get_sessions returned error: %v", res.Content)
	}

	// Should have at least one content block.
	if len(res.Content) == 0 {
		t.Fatal("expected content from get_sessions, got none")
	}

	// First content should be text.
	tc, ok := res.Content[0].(*sdkmcp.TextContent)
	if !ok {
		t.Fatalf("expected TextContent, got %T", res.Content[0])
	}

	// The text should be valid JSON (an array).
	var sessions []json.RawMessage
	if err := json.Unmarshal([]byte(tc.Text), &sessions); err != nil {
		t.Errorf("get_sessions returned invalid JSON: %v", err)
	}

	cancel()
	<-errCh
}

// ---------------------------------------------------------------------------
// TestGetBriefTool -- calling get_brief returns formatted text.
// ---------------------------------------------------------------------------

func TestGetBriefTool(t *testing.T) {
	srv := NewServer()

	client := sdkmcp.NewClient(&sdkmcp.Implementation{
		Name:    "test-client",
		Version: "0.0.1",
	}, nil)

	st, ct := sdkmcp.NewInMemoryTransports()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	errCh := make(chan error, 1)
	go func() { errCh <- srv.Run(ctx, st) }()

	session, err := client.Connect(ctx, ct, nil)
	if err != nil {
		t.Fatalf("client.Connect failed: %v", err)
	}
	defer session.Close()

	res, err := session.CallTool(ctx, &sdkmcp.CallToolParams{
		Name:      "get_brief",
		Arguments: map[string]any{"since": "24h"},
	})
	if err != nil {
		t.Fatalf("CallTool(get_brief) failed: %v", err)
	}

	if res.IsError {
		t.Fatalf("get_brief returned error: %v", res.Content)
	}

	if len(res.Content) == 0 {
		t.Fatal("expected content from get_brief, got none")
	}

	tc, ok := res.Content[0].(*sdkmcp.TextContent)
	if !ok {
		t.Fatalf("expected TextContent, got %T", res.Content[0])
	}

	// Should contain some text (even if no sessions found).
	if tc.Text == "" {
		t.Error("get_brief returned empty text")
	}

	cancel()
	<-errCh
}

// ---------------------------------------------------------------------------
// TestGetBriefToolNoParam -- calling get_brief without since uses default.
// ---------------------------------------------------------------------------

func TestGetBriefToolNoParam(t *testing.T) {
	srv := NewServer()

	client := sdkmcp.NewClient(&sdkmcp.Implementation{
		Name:    "test-client",
		Version: "0.0.1",
	}, nil)

	st, ct := sdkmcp.NewInMemoryTransports()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	errCh := make(chan error, 1)
	go func() { errCh <- srv.Run(ctx, st) }()

	session, err := client.Connect(ctx, ct, nil)
	if err != nil {
		t.Fatalf("client.Connect failed: %v", err)
	}
	defer session.Close()

	// No arguments at all.
	res, err := session.CallTool(ctx, &sdkmcp.CallToolParams{
		Name: "get_brief",
	})
	if err != nil {
		t.Fatalf("CallTool(get_brief) failed: %v", err)
	}

	if res.IsError {
		t.Fatalf("get_brief (no param) returned error: %v", res.Content)
	}

	if len(res.Content) == 0 {
		t.Fatal("expected content from get_brief (no param)")
	}

	cancel()
	<-errCh
}

// ---------------------------------------------------------------------------
// TestGetCostsTool -- calling get_costs returns formatted text.
// ---------------------------------------------------------------------------

func TestGetCostsTool(t *testing.T) {
	srv := NewServer()

	client := sdkmcp.NewClient(&sdkmcp.Implementation{
		Name:    "test-client",
		Version: "0.0.1",
	}, nil)

	st, ct := sdkmcp.NewInMemoryTransports()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	errCh := make(chan error, 1)
	go func() { errCh <- srv.Run(ctx, st) }()

	session, err := client.Connect(ctx, ct, nil)
	if err != nil {
		t.Fatalf("client.Connect failed: %v", err)
	}
	defer session.Close()

	res, err := session.CallTool(ctx, &sdkmcp.CallToolParams{
		Name:      "get_costs",
		Arguments: map[string]any{"since": "24h"},
	})
	if err != nil {
		t.Fatalf("CallTool(get_costs) failed: %v", err)
	}

	if res.IsError {
		t.Fatalf("get_costs returned error: %v", res.Content)
	}

	if len(res.Content) == 0 {
		t.Fatal("expected content from get_costs")
	}

	tc, ok := res.Content[0].(*sdkmcp.TextContent)
	if !ok {
		t.Fatalf("expected TextContent, got %T", res.Content[0])
	}

	if tc.Text == "" {
		t.Error("get_costs returned empty text")
	}

	cancel()
	<-errCh
}

// ---------------------------------------------------------------------------
// TestGetSessionDetailToolMissing -- calling with a nonexistent session ID.
// The tool should return an error result (not a protocol error).
// ---------------------------------------------------------------------------

func TestGetSessionDetailToolMissing(t *testing.T) {
	srv := NewServer()

	client := sdkmcp.NewClient(&sdkmcp.Implementation{
		Name:    "test-client",
		Version: "0.0.1",
	}, nil)

	st, ct := sdkmcp.NewInMemoryTransports()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	errCh := make(chan error, 1)
	go func() { errCh <- srv.Run(ctx, st) }()

	session, err := client.Connect(ctx, ct, nil)
	if err != nil {
		t.Fatalf("client.Connect failed: %v", err)
	}
	defer session.Close()

	res, err := session.CallTool(ctx, &sdkmcp.CallToolParams{
		Name:      "get_session_detail",
		Arguments: map[string]any{"session_id": "nonexistent-id"},
	})
	if err != nil {
		t.Fatalf("CallTool(get_session_detail) failed: %v", err)
	}

	// Should be a tool-level error (not found), not a protocol error.
	if !res.IsError {
		t.Error("expected IsError=true for nonexistent session")
	}

	cancel()
	<-errCh
}

// ---------------------------------------------------------------------------
// TestParseSinceDuration -- the helper that converts "24h" -> time.Time.
// ---------------------------------------------------------------------------

func TestParseSinceDuration(t *testing.T) {
	cases := []struct {
		input    string
		wantBack time.Duration
	}{
		{"24h", 24 * time.Hour},
		{"12h", 12 * time.Hour},
		{"1h", 1 * time.Hour},
		{"", 24 * time.Hour}, // default
	}

	for _, tc := range cases {
		since := parseSinceDuration(tc.input)
		elapsed := time.Since(since)
		// Allow 1 second of slop.
		if elapsed < tc.wantBack-time.Second || elapsed > tc.wantBack+time.Second {
			t.Errorf("parseSinceDuration(%q): elapsed=%v, want ~%v", tc.input, elapsed, tc.wantBack)
		}
	}
}

// ---------------------------------------------------------------------------
// TestGetSessionDetailToolNoArg -- calling without session_id should fail.
// The SDK validates the input schema, so we expect a protocol error.
// ---------------------------------------------------------------------------

func TestGetSessionDetailToolNoArg(t *testing.T) {
	srv := NewServer()

	client := sdkmcp.NewClient(&sdkmcp.Implementation{
		Name:    "test-client",
		Version: "0.0.1",
	}, nil)

	st, ct := sdkmcp.NewInMemoryTransports()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	errCh := make(chan error, 1)
	go func() { errCh <- srv.Run(ctx, st) }()

	session, err := client.Connect(ctx, ct, nil)
	if err != nil {
		t.Fatalf("client.Connect failed: %v", err)
	}
	defer session.Close()

	// Call without the required session_id argument.
	res, err := session.CallTool(ctx, &sdkmcp.CallToolParams{
		Name: "get_session_detail",
	})

	// The SDK should return a validation error (either as protocol error or tool error).
	if err == nil && res != nil && !res.IsError {
		t.Error("expected error when calling get_session_detail without session_id")
	}

	cancel()
	<-errCh
}
