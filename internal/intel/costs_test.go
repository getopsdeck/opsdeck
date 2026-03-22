package intel

import (
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// JSONL lines with usage data (assistant messages containing model + usage)
// ---------------------------------------------------------------------------

const (
	// Assistant message with Opus model and usage data.
	assistantOpusUsageLine = `{"type":"assistant","message":{"role":"assistant","model":"claude-opus-4-6","content":[{"type":"text","text":"Here is the fix."}],"usage":{"input_tokens":5000,"output_tokens":2000,"cache_creation_input_tokens":1000,"cache_read_input_tokens":500}},"timestamp":"2026-03-21T10:01:00.000Z","uuid":"a-opus","sessionId":"sess-cost-001","cwd":"/home/dev/costproject"}`

	// Assistant message with Sonnet model and usage data.
	assistantSonnetUsageLine = `{"type":"assistant","message":{"role":"assistant","model":"claude-sonnet-4-6","content":[{"type":"text","text":"Done."}],"usage":{"input_tokens":3000,"output_tokens":1000,"cache_creation_input_tokens":0,"cache_read_input_tokens":200}},"timestamp":"2026-03-21T10:02:00.000Z","uuid":"a-sonnet","sessionId":"sess-cost-001","cwd":"/home/dev/costproject"}`

	// Assistant message with Haiku model and usage data.
	assistantHaikuUsageLine = `{"type":"assistant","message":{"role":"assistant","model":"claude-haiku-4-5","content":[{"type":"text","text":"Quick answer."}],"usage":{"input_tokens":1000,"output_tokens":500,"cache_creation_input_tokens":0,"cache_read_input_tokens":0}},"timestamp":"2026-03-21T10:03:00.000Z","uuid":"a-haiku","sessionId":"sess-cost-002","cwd":"/home/dev/costproject"}`

	// Assistant message with unknown model.
	assistantUnknownModelLine = `{"type":"assistant","message":{"role":"assistant","model":"claude-mystery-9","content":[{"type":"text","text":"Mystery model."}],"usage":{"input_tokens":2000,"output_tokens":1000,"cache_creation_input_tokens":0,"cache_read_input_tokens":0}},"timestamp":"2026-03-21T10:04:00.000Z","uuid":"a-unknown","sessionId":"sess-cost-003","cwd":"/home/dev/costproject"}`

	// User message (no usage data -- should be skipped for cost extraction).
	userMsgCostLine = `{"type":"user","message":{"role":"user","content":"Estimate my costs"},"isMeta":false,"timestamp":"2026-03-21T10:00:00.000Z","uuid":"u-cost","sessionId":"sess-cost-001","cwd":"/home/dev/costproject"}`

	// Assistant message with zero tokens (should be skipped).
	assistantZeroUsageLine = `{"type":"assistant","message":{"role":"assistant","model":"claude-sonnet-4-6","content":[{"type":"text","text":"No tokens?"}],"usage":{"input_tokens":0,"output_tokens":0,"cache_creation_input_tokens":0,"cache_read_input_tokens":0}},"timestamp":"2026-03-21T10:01:30.000Z","uuid":"a-zero","sessionId":"sess-cost-001","cwd":"/home/dev/costproject"}`

	// Assistant message without usage block at all.
	assistantNoUsageLine = `{"type":"assistant","message":{"role":"assistant","model":"claude-sonnet-4-6","content":[{"type":"text","text":"No usage block."}]},"timestamp":"2026-03-21T10:01:45.000Z","uuid":"a-nousage","sessionId":"sess-cost-001","cwd":"/home/dev/costproject"}`

	// Early message for time-filtering tests.
	assistantEarlyLine = `{"type":"assistant","message":{"role":"assistant","model":"claude-sonnet-4-6","content":[{"type":"text","text":"Early work."}],"usage":{"input_tokens":1000,"output_tokens":500,"cache_creation_input_tokens":0,"cache_read_input_tokens":0}},"timestamp":"2026-03-21T08:00:00.000Z","uuid":"a-early","sessionId":"sess-cost-004","cwd":"/home/dev/timeproject"}`

	// Late message for time-filtering tests.
	assistantLateLine = `{"type":"assistant","message":{"role":"assistant","model":"claude-sonnet-4-6","content":[{"type":"text","text":"Late work."}],"usage":{"input_tokens":2000,"output_tokens":800,"cache_creation_input_tokens":100,"cache_read_input_tokens":50}},"timestamp":"2026-03-21T14:00:00.000Z","uuid":"a-late","sessionId":"sess-cost-004","cwd":"/home/dev/timeproject"}`

	// User message with timestamp for time-filtering context.
	userMsgEarlyLine = `{"type":"user","message":{"role":"user","content":"Start work"},"isMeta":false,"timestamp":"2026-03-21T07:59:00.000Z","uuid":"u-early","sessionId":"sess-cost-004","cwd":"/home/dev/timeproject"}`
)

// ---------------------------------------------------------------------------
// ExtractCosts tests
// ---------------------------------------------------------------------------

func TestExtractCosts_SingleOpusMessage(t *testing.T) {
	dir := t.TempDir()
	path := writeTranscript(t, dir,
		userMsgCostLine,
		assistantOpusUsageLine,
	)

	cost, err := ExtractCosts(path, time.Time{})
	if err != nil {
		t.Fatalf("ExtractCosts() error = %v", err)
	}

	if cost.SessionID != "sess-cost-001" {
		t.Errorf("SessionID = %q, want %q", cost.SessionID, "sess-cost-001")
	}
	if cost.Project != "costproject" {
		t.Errorf("Project = %q, want %q", cost.Project, "costproject")
	}
	if cost.Model != "claude-opus-4-6" {
		t.Errorf("Model = %q, want %q", cost.Model, "claude-opus-4-6")
	}
	if cost.InputTokens != 5000 {
		t.Errorf("InputTokens = %d, want 5000", cost.InputTokens)
	}
	if cost.OutputTokens != 2000 {
		t.Errorf("OutputTokens = %d, want 2000", cost.OutputTokens)
	}
	if cost.CacheWrite != 1000 {
		t.Errorf("CacheWrite = %d, want 1000", cost.CacheWrite)
	}
	if cost.CacheRead != 500 {
		t.Errorf("CacheRead = %d, want 500", cost.CacheRead)
	}
	if cost.TotalTokens != 8500 {
		t.Errorf("TotalTokens = %d, want 8500", cost.TotalTokens)
	}
	if cost.MessageCount != 1 {
		t.Errorf("MessageCount = %d, want 1", cost.MessageCount)
	}
	if cost.EstCostUSD <= 0 {
		t.Errorf("EstCostUSD = %f, want > 0", cost.EstCostUSD)
	}
}

func TestExtractCosts_MultipleMessages(t *testing.T) {
	dir := t.TempDir()
	path := writeTranscript(t, dir,
		userMsgCostLine,
		assistantOpusUsageLine,
		assistantSonnetUsageLine,
	)

	cost, err := ExtractCosts(path, time.Time{})
	if err != nil {
		t.Fatalf("ExtractCosts() error = %v", err)
	}

	// Tokens should be summed across messages.
	wantInput := int64(5000 + 3000)
	if cost.InputTokens != wantInput {
		t.Errorf("InputTokens = %d, want %d", cost.InputTokens, wantInput)
	}
	wantOutput := int64(2000 + 1000)
	if cost.OutputTokens != wantOutput {
		t.Errorf("OutputTokens = %d, want %d", cost.OutputTokens, wantOutput)
	}
	wantCacheWrite := int64(1000 + 0)
	if cost.CacheWrite != wantCacheWrite {
		t.Errorf("CacheWrite = %d, want %d", cost.CacheWrite, wantCacheWrite)
	}
	wantCacheRead := int64(500 + 200)
	if cost.CacheRead != wantCacheRead {
		t.Errorf("CacheRead = %d, want %d", cost.CacheRead, wantCacheRead)
	}
	wantTotal := wantInput + wantOutput + wantCacheWrite + wantCacheRead
	if cost.TotalTokens != wantTotal {
		t.Errorf("TotalTokens = %d, want %d", cost.TotalTokens, wantTotal)
	}
	if cost.MessageCount != 2 {
		t.Errorf("MessageCount = %d, want 2", cost.MessageCount)
	}

	// Model should be the last one seen.
	if cost.Model != "claude-sonnet-4-6" {
		t.Errorf("Model = %q, want %q (last model seen)", cost.Model, "claude-sonnet-4-6")
	}
}

func TestExtractCosts_SkipsZeroUsage(t *testing.T) {
	dir := t.TempDir()
	path := writeTranscript(t, dir,
		userMsgCostLine,
		assistantZeroUsageLine,
		assistantOpusUsageLine,
	)

	cost, err := ExtractCosts(path, time.Time{})
	if err != nil {
		t.Fatalf("ExtractCosts() error = %v", err)
	}

	// Zero-usage message should be skipped.
	if cost.MessageCount != 1 {
		t.Errorf("MessageCount = %d, want 1 (zero-usage should be skipped)", cost.MessageCount)
	}
	if cost.InputTokens != 5000 {
		t.Errorf("InputTokens = %d, want 5000", cost.InputTokens)
	}
}

func TestExtractCosts_SkipsNoUsageBlock(t *testing.T) {
	dir := t.TempDir()
	path := writeTranscript(t, dir,
		assistantNoUsageLine,
		assistantOpusUsageLine,
	)

	cost, err := ExtractCosts(path, time.Time{})
	if err != nil {
		t.Fatalf("ExtractCosts() error = %v", err)
	}

	// Message without usage block has zero tokens, so it should be skipped.
	if cost.MessageCount != 1 {
		t.Errorf("MessageCount = %d, want 1", cost.MessageCount)
	}
}

func TestExtractCosts_SkipsUserMessages(t *testing.T) {
	dir := t.TempDir()
	path := writeTranscript(t, dir,
		userMsgCostLine,
	)

	cost, err := ExtractCosts(path, time.Time{})
	if err != nil {
		t.Fatalf("ExtractCosts() error = %v", err)
	}

	if cost.MessageCount != 0 {
		t.Errorf("MessageCount = %d, want 0 (user messages have no usage)", cost.MessageCount)
	}
	if cost.TotalTokens != 0 {
		t.Errorf("TotalTokens = %d, want 0", cost.TotalTokens)
	}
}

func TestExtractCosts_MalformedLines(t *testing.T) {
	dir := t.TempDir()
	path := writeTranscript(t, dir,
		`{broken json!!!`,
		`not json at all`,
		assistantOpusUsageLine,
		``,
	)

	cost, err := ExtractCosts(path, time.Time{})
	if err != nil {
		t.Fatalf("ExtractCosts() error = %v", err)
	}

	// Should still extract from the valid line.
	if cost.MessageCount != 1 {
		t.Errorf("MessageCount = %d, want 1", cost.MessageCount)
	}
	if cost.InputTokens != 5000 {
		t.Errorf("InputTokens = %d, want 5000", cost.InputTokens)
	}
}

func TestExtractCosts_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty.jsonl")
	if err := os.WriteFile(path, []byte(""), 0644); err != nil {
		t.Fatal(err)
	}

	cost, err := ExtractCosts(path, time.Time{})
	if err != nil {
		t.Fatalf("ExtractCosts() error = %v", err)
	}

	if cost.TotalTokens != 0 {
		t.Errorf("TotalTokens = %d, want 0", cost.TotalTokens)
	}
	if cost.MessageCount != 0 {
		t.Errorf("MessageCount = %d, want 0", cost.MessageCount)
	}
}

// ---------------------------------------------------------------------------
// ExtractCosts with time filtering (since parameter)
// ---------------------------------------------------------------------------

func TestExtractCosts_FilterBySince(t *testing.T) {
	dir := t.TempDir()
	path := writeTranscript(t, dir,
		userMsgEarlyLine,  // 07:59 - user msg
		assistantEarlyLine, // 08:00 - should be filtered out
		assistantLateLine,  // 14:00 - should be included
	)

	// Filter: only after 10:00.
	since := mustParseTime("2026-03-21T10:00:00.000Z")
	cost, err := ExtractCosts(path, since)
	if err != nil {
		t.Fatalf("ExtractCosts() error = %v", err)
	}

	// Only the late message (14:00) should be counted.
	if cost.MessageCount != 1 {
		t.Errorf("MessageCount = %d, want 1 (only late message)", cost.MessageCount)
	}
	if cost.InputTokens != 2000 {
		t.Errorf("InputTokens = %d, want 2000", cost.InputTokens)
	}
	if cost.OutputTokens != 800 {
		t.Errorf("OutputTokens = %d, want 800", cost.OutputTokens)
	}
	if cost.CacheWrite != 100 {
		t.Errorf("CacheWrite = %d, want 100", cost.CacheWrite)
	}
	if cost.CacheRead != 50 {
		t.Errorf("CacheRead = %d, want 50", cost.CacheRead)
	}
}

func TestExtractCosts_ZeroSinceIncludesAll(t *testing.T) {
	dir := t.TempDir()
	path := writeTranscript(t, dir,
		assistantEarlyLine, // 08:00
		assistantLateLine,  // 14:00
	)

	// Zero time means no filtering.
	cost, err := ExtractCosts(path, time.Time{})
	if err != nil {
		t.Fatalf("ExtractCosts() error = %v", err)
	}

	if cost.MessageCount != 2 {
		t.Errorf("MessageCount = %d, want 2 (no filtering)", cost.MessageCount)
	}
	wantInput := int64(1000 + 2000)
	if cost.InputTokens != wantInput {
		t.Errorf("InputTokens = %d, want %d", cost.InputTokens, wantInput)
	}
}

func TestExtractCosts_SinceAfterAllMessages(t *testing.T) {
	dir := t.TempDir()
	path := writeTranscript(t, dir,
		assistantEarlyLine, // 08:00
		assistantLateLine,  // 14:00
	)

	// Filter: after 23:00 (nothing qualifies).
	since := mustParseTime("2026-03-21T23:00:00.000Z")
	cost, err := ExtractCosts(path, since)
	if err != nil {
		t.Fatalf("ExtractCosts() error = %v", err)
	}

	if cost.MessageCount != 0 {
		t.Errorf("MessageCount = %d, want 0", cost.MessageCount)
	}
	if cost.TotalTokens != 0 {
		t.Errorf("TotalTokens = %d, want 0", cost.TotalTokens)
	}
}

// ---------------------------------------------------------------------------
// ExtractCosts with missing file
// ---------------------------------------------------------------------------

func TestExtractCosts_MissingFile(t *testing.T) {
	_, err := ExtractCosts("/nonexistent/path/transcript.jsonl", time.Time{})
	if err == nil {
		t.Fatal("ExtractCosts() should error on missing file")
	}
}

// ---------------------------------------------------------------------------
// estimateCostForMessage tests
// ---------------------------------------------------------------------------

// testEstimateCost is a helper that adapts the old SessionCost-based tests
// to the new per-message estimateCostForMessage function.
func testEstimateCost(c SessionCost) float64 {
	return estimateCostForMessage(c.Model, transcriptUsage{
		InputTokens:              c.InputTokens,
		OutputTokens:             c.OutputTokens,
		CacheCreationInputTokens: c.CacheWrite,
		CacheReadInputTokens:     c.CacheRead,
	})
}

func TestEstimateCost_OpusPricing(t *testing.T) {
	c := SessionCost{
		Model:        "claude-opus-4-6",
		InputTokens:  1_000_000,
		OutputTokens: 1_000_000,
		CacheWrite:   1_000_000,
		CacheRead:    1_000_000,
	}

	got := testEstimateCost(c)
	// Opus: input=15, output=75, cache_write=18.75, cache_read=1.50
	want := 15.0 + 75.0 + 18.75 + 1.50
	if math.Abs(got-want) > 0.001 {
		t.Errorf("estimateCost(opus) = %.4f, want %.4f", got, want)
	}
}

func TestEstimateCost_SonnetPricing(t *testing.T) {
	c := SessionCost{
		Model:        "claude-sonnet-4-6",
		InputTokens:  1_000_000,
		OutputTokens: 1_000_000,
		CacheWrite:   1_000_000,
		CacheRead:    1_000_000,
	}

	got := testEstimateCost(c)
	// Sonnet: input=3, output=15, cache_write=3.75, cache_read=0.30
	want := 3.0 + 15.0 + 3.75 + 0.30
	if math.Abs(got-want) > 0.001 {
		t.Errorf("estimateCost(sonnet) = %.4f, want %.4f", got, want)
	}
}

func TestEstimateCost_HaikuPricing(t *testing.T) {
	c := SessionCost{
		Model:        "claude-haiku-4-5",
		InputTokens:  1_000_000,
		OutputTokens: 1_000_000,
		CacheWrite:   1_000_000,
		CacheRead:    1_000_000,
	}

	got := testEstimateCost(c)
	// Haiku: input=0.80, output=4.0, cache_write=1.0, cache_read=0.08
	want := 0.80 + 4.0 + 1.0 + 0.08
	if math.Abs(got-want) > 0.001 {
		t.Errorf("estimateCost(haiku) = %.4f, want %.4f", got, want)
	}
}

func TestEstimateCost_UnknownModelDefaultsToSonnet(t *testing.T) {
	c := SessionCost{
		Model:        "claude-mystery-9",
		InputTokens:  1_000_000,
		OutputTokens: 1_000_000,
		CacheWrite:   1_000_000,
		CacheRead:    1_000_000,
	}

	got := testEstimateCost(c)
	// Should fall back to Sonnet pricing.
	want := 3.0 + 15.0 + 3.75 + 0.30
	if math.Abs(got-want) > 0.001 {
		t.Errorf("estimateCost(unknown) = %.4f, want %.4f (Sonnet fallback)", got, want)
	}
}

func TestEstimateCost_EmptyModelDefaultsToSonnet(t *testing.T) {
	c := SessionCost{
		Model:        "",
		InputTokens:  1_000_000,
		OutputTokens: 1_000_000,
		CacheWrite:   0,
		CacheRead:    0,
	}

	got := testEstimateCost(c)
	want := 3.0 + 15.0 // input + output only, Sonnet pricing
	if math.Abs(got-want) > 0.001 {
		t.Errorf("estimateCost(empty model) = %.4f, want %.4f", got, want)
	}
}

func TestEstimateCost_ZeroTokens(t *testing.T) {
	c := SessionCost{Model: "claude-opus-4-6"}
	got := testEstimateCost(c)
	if got != 0 {
		t.Errorf("estimateCost(zero tokens) = %f, want 0", got)
	}
}

func TestEstimateCost_OnlyInputTokens(t *testing.T) {
	c := SessionCost{
		Model:       "claude-opus-4-6",
		InputTokens: 500_000,
	}

	got := testEstimateCost(c)
	want := 500_000.0 * 15.0 / 1_000_000 // 7.5
	if math.Abs(got-want) > 0.001 {
		t.Errorf("estimateCost(input only) = %.4f, want %.4f", got, want)
	}
}

func TestEstimateCost_OnlyCacheTokens(t *testing.T) {
	c := SessionCost{
		Model:      "claude-opus-4-6",
		CacheWrite: 200_000,
		CacheRead:  800_000,
	}

	got := testEstimateCost(c)
	wantWrite := 200_000.0 * 18.75 / 1_000_000
	wantRead := 800_000.0 * 1.50 / 1_000_000
	want := wantWrite + wantRead
	if math.Abs(got-want) > 0.001 {
		t.Errorf("estimateCost(cache only) = %.4f, want %.4f", got, want)
	}
}

// ---------------------------------------------------------------------------
// formatTokens tests
// ---------------------------------------------------------------------------

func TestFormatTokens_Millions(t *testing.T) {
	tests := []struct {
		input int64
		want  string
	}{
		{1_000_000, "1.0M"},
		{1_500_000, "1.5M"},
		{2_345_678, "2.3M"},
		{10_000_000, "10.0M"},
	}
	for _, tc := range tests {
		got := formatTokens(tc.input)
		if got != tc.want {
			t.Errorf("formatTokens(%d) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestFormatTokens_Thousands(t *testing.T) {
	tests := []struct {
		input int64
		want  string
	}{
		{1_000, "1.0K"},
		{1_500, "1.5K"},
		{999_999, "1000.0K"},
		{50_000, "50.0K"},
	}
	for _, tc := range tests {
		got := formatTokens(tc.input)
		if got != tc.want {
			t.Errorf("formatTokens(%d) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestFormatTokens_Small(t *testing.T) {
	tests := []struct {
		input int64
		want  string
	}{
		{0, "0"},
		{1, "1"},
		{999, "999"},
		{500, "500"},
	}
	for _, tc := range tests {
		got := formatTokens(tc.input)
		if got != tc.want {
			t.Errorf("formatTokens(%d) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

// ---------------------------------------------------------------------------
// GenerateCostReport tests
// ---------------------------------------------------------------------------

func TestGenerateCostReport_WithMockTranscripts(t *testing.T) {
	// Set up a projects directory with two project subdirectories.
	projectsDir := t.TempDir()

	// Project A: one session with Opus usage.
	projectADir := filepath.Join(projectsDir, "project-a")
	if err := os.MkdirAll(projectADir, 0755); err != nil {
		t.Fatal(err)
	}
	writeTranscriptTo(t, filepath.Join(projectADir, "session-1.jsonl"),
		userMsgCostLine,
		assistantOpusUsageLine,
	)

	// Project B: one session with Sonnet usage.
	projectBDir := filepath.Join(projectsDir, "project-b")
	if err := os.MkdirAll(projectBDir, 0755); err != nil {
		t.Fatal(err)
	}
	writeTranscriptTo(t, filepath.Join(projectBDir, "session-2.jsonl"),
		assistantSonnetUsageLine,
	)

	report, err := GenerateCostReport(projectsDir, "", time.Time{})
	if err != nil {
		t.Fatalf("GenerateCostReport() error = %v", err)
	}

	if len(report.Sessions) != 2 {
		t.Fatalf("Sessions count = %d, want 2", len(report.Sessions))
	}
	if report.TotalCost <= 0 {
		t.Errorf("TotalCost = %f, want > 0", report.TotalCost)
	}
	if report.TotalTokens <= 0 {
		t.Errorf("TotalTokens = %d, want > 0", report.TotalTokens)
	}
	if report.Period != "All time" {
		t.Errorf("Period = %q, want %q", report.Period, "All time")
	}

	// Sessions should be sorted by cost descending (Opus is more expensive).
	if report.Sessions[0].EstCostUSD < report.Sessions[1].EstCostUSD {
		t.Errorf("Sessions not sorted by cost descending: first=$%.4f, second=$%.4f",
			report.Sessions[0].EstCostUSD, report.Sessions[1].EstCostUSD)
	}
}

func TestGenerateCostReport_WithSinceFilter(t *testing.T) {
	projectsDir := t.TempDir()

	projectDir := filepath.Join(projectsDir, "timeproject")
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatal(err)
	}
	writeTranscriptTo(t, filepath.Join(projectDir, "session-time.jsonl"),
		userMsgEarlyLine,   // 07:59
		assistantEarlyLine, // 08:00
		assistantLateLine,  // 14:00
	)

	// Only include messages after 10:00.
	since := mustParseTime("2026-03-21T10:00:00.000Z")
	report, err := GenerateCostReport(projectsDir, "", since)
	if err != nil {
		t.Fatalf("GenerateCostReport() error = %v", err)
	}

	if len(report.Sessions) != 1 {
		t.Fatalf("Sessions count = %d, want 1", len(report.Sessions))
	}

	// Only the late message tokens should be counted.
	session := report.Sessions[0]
	if session.InputTokens != 2000 {
		t.Errorf("InputTokens = %d, want 2000 (only late message)", session.InputTokens)
	}

	if !strings.Contains(report.Period, "Since") {
		t.Errorf("Period = %q, want it to contain 'Since'", report.Period)
	}
}

func TestGenerateCostReport_EmptyProjectsDir(t *testing.T) {
	projectsDir := t.TempDir()

	report, err := GenerateCostReport(projectsDir, "", time.Time{})
	if err != nil {
		t.Fatalf("GenerateCostReport() error = %v", err)
	}

	if len(report.Sessions) != 0 {
		t.Errorf("Sessions count = %d, want 0", len(report.Sessions))
	}
	if report.TotalCost != 0 {
		t.Errorf("TotalCost = %f, want 0", report.TotalCost)
	}
}

func TestGenerateCostReport_NonexistentDir(t *testing.T) {
	report, err := GenerateCostReport("/nonexistent/dir", "", time.Time{})
	if err != nil {
		t.Fatalf("GenerateCostReport() should not error on nonexistent dir, got %v", err)
	}

	if len(report.Sessions) != 0 {
		t.Errorf("Sessions count = %d, want 0", len(report.Sessions))
	}
}

func TestGenerateCostReport_SkipsNonJSONLFiles(t *testing.T) {
	projectsDir := t.TempDir()

	projectDir := filepath.Join(projectsDir, "proj")
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Write a non-JSONL file (should be ignored).
	if err := os.WriteFile(filepath.Join(projectDir, "notes.txt"), []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}

	// Write a valid JSONL file.
	writeTranscriptTo(t, filepath.Join(projectDir, "session.jsonl"),
		assistantOpusUsageLine,
	)

	report, err := GenerateCostReport(projectsDir, "", time.Time{})
	if err != nil {
		t.Fatalf("GenerateCostReport() error = %v", err)
	}

	if len(report.Sessions) != 1 {
		t.Errorf("Sessions count = %d, want 1 (only .jsonl)", len(report.Sessions))
	}
}

func TestGenerateCostReport_SkipsZeroTokenSessions(t *testing.T) {
	projectsDir := t.TempDir()

	projectDir := filepath.Join(projectsDir, "proj")
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Session with only user messages (zero tokens).
	writeTranscriptTo(t, filepath.Join(projectDir, "empty-session.jsonl"),
		userMsgCostLine,
	)

	// Session with actual usage.
	writeTranscriptTo(t, filepath.Join(projectDir, "real-session.jsonl"),
		assistantHaikuUsageLine,
	)

	report, err := GenerateCostReport(projectsDir, "", time.Time{})
	if err != nil {
		t.Fatalf("GenerateCostReport() error = %v", err)
	}

	if len(report.Sessions) != 1 {
		t.Errorf("Sessions count = %d, want 1 (zero-token sessions skipped)", len(report.Sessions))
	}
}

// ---------------------------------------------------------------------------
// FormatCostReport tests
// ---------------------------------------------------------------------------

func TestFormatCostReport_BasicOutput(t *testing.T) {
	report := CostReport{
		Sessions: []SessionCost{
			{
				SessionID:    "sess-abc-123-456",
				Project:      "myproject",
				Model:        "claude-opus-4-6",
				InputTokens:  50000,
				OutputTokens: 20000,
				TotalTokens:  70000,
				EstCostUSD:   2.25,
				MessageCount: 10,
			},
		},
		TotalCost:   2.25,
		TotalTokens: 70000,
		Period:      "All time",
	}

	output := FormatCostReport(report)

	if !strings.Contains(output, "OpsDeck Cost Report") {
		t.Errorf("output should contain header")
	}
	if !strings.Contains(output, "All time") {
		t.Errorf("output should contain period")
	}
	if !strings.Contains(output, "$2.25") {
		t.Errorf("output should contain total cost")
	}
	if !strings.Contains(output, "myproject") {
		t.Errorf("output should contain project name")
	}
	if !strings.Contains(output, "sess-abc-123") {
		t.Errorf("output should contain truncated session ID (max 12 chars)")
	}
	if !strings.Contains(output, "claude-opus-4-6") {
		t.Errorf("output should contain model name")
	}
	if !strings.Contains(output, "70.0K") {
		t.Errorf("output should contain formatted token count")
	}
}

func TestFormatCostReport_GroupsByProject(t *testing.T) {
	report := CostReport{
		Sessions: []SessionCost{
			{SessionID: "s1", Project: "alpha", Model: "claude-opus-4-6", EstCostUSD: 5.0, TotalTokens: 100000},
			{SessionID: "s2", Project: "beta", Model: "claude-sonnet-4-6", EstCostUSD: 1.0, TotalTokens: 50000},
			{SessionID: "s3", Project: "alpha", Model: "claude-opus-4-6", EstCostUSD: 3.0, TotalTokens: 80000},
		},
		TotalCost:   9.0,
		TotalTokens: 230000,
		Period:      "All time",
	}

	output := FormatCostReport(report)

	// Both projects should appear.
	if !strings.Contains(output, "alpha") {
		t.Errorf("output should contain project 'alpha'")
	}
	if !strings.Contains(output, "beta") {
		t.Errorf("output should contain project 'beta'")
	}

	// Alpha ($8.00 total) should appear before Beta ($1.00).
	alphaIdx := strings.Index(output, "alpha")
	betaIdx := strings.Index(output, "beta")
	if alphaIdx > betaIdx {
		t.Errorf("alpha ($8) should appear before beta ($1) in output")
	}
}

func TestFormatCostReport_EmptyReport(t *testing.T) {
	report := CostReport{
		Period: "All time",
	}

	output := FormatCostReport(report)

	if !strings.Contains(output, "OpsDeck Cost Report") {
		t.Error("empty report should still have header")
	}
	if !strings.Contains(output, "$0.00") {
		t.Errorf("empty report should show $0.00, got: %s", output)
	}
	if !strings.Contains(output, "0 sessions") {
		t.Errorf("empty report should show 0 sessions, got: %s", output)
	}
}

func TestFormatCostReport_ShortSessionID(t *testing.T) {
	report := CostReport{
		Sessions: []SessionCost{
			{SessionID: "short", Project: "proj", Model: "claude-sonnet-4-6", TotalTokens: 1000, EstCostUSD: 0.01},
		},
		TotalCost:   0.01,
		TotalTokens: 1000,
		Period:      "All time",
	}

	output := FormatCostReport(report)

	// Short IDs should not be truncated.
	if !strings.Contains(output, "short") {
		t.Errorf("short session ID should appear as-is, got: %s", output)
	}
}

// ---------------------------------------------------------------------------
// Integration: ExtractCosts + estimateCost consistency
// ---------------------------------------------------------------------------

func TestExtractCosts_CostMatchesManualCalculation(t *testing.T) {
	dir := t.TempDir()
	path := writeTranscript(t, dir,
		assistantOpusUsageLine,
	)

	cost, err := ExtractCosts(path, time.Time{})
	if err != nil {
		t.Fatalf("ExtractCosts() error = %v", err)
	}

	// Manual calculation for Opus with tokens: input=5000, output=2000, cache_write=1000, cache_read=500
	wantCost := 5000.0*15.0/1_000_000 + 2000.0*75.0/1_000_000 + 1000.0*18.75/1_000_000 + 500.0*1.50/1_000_000
	if math.Abs(cost.EstCostUSD-wantCost) > 0.0001 {
		t.Errorf("EstCostUSD = %.6f, want %.6f", cost.EstCostUSD, wantCost)
	}
}

// ---------------------------------------------------------------------------
// Helper: write transcript to a specific path (not random name)
// ---------------------------------------------------------------------------

func writeTranscriptTo(t *testing.T, path string, lines ...string) {
	t.Helper()
	data := strings.Join(lines, "\n") + "\n"
	if err := os.WriteFile(path, []byte(data), 0644); err != nil {
		t.Fatal(err)
	}
}
