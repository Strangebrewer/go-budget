package tracer

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
)

type Client struct {
	url        string
	serviceKey string
	service    string
	http       *http.Client
}

func NewClient(url, serviceKey string, service string) *Client {
	return &Client{
		url:        url,
		serviceKey: serviceKey,
		service:    service,
		http:       &http.Client{Timeout: 5 * time.Second},
	}
}

type Span struct {
	TraceID      string         `json:"traceId"`
	SpanID       string         `json:"spanId"`
	ParentSpanID string         `json:"parentSpanId,omitempty"`
	Service      string         `json:"service"`
	Operation    string         `json:"operation"`
	Status       string         `json:"status"`
	Error        *string        `json:"error,omitempty"`
	StartTime    time.Time      `json:"startTime"`
	EndTime      time.Time      `json:"endTime"`
	Metadata     map[string]any `json:"metadata,omitempty"`
}

// Send fires the span to go-tracer in a goroutine. Errors are logged but never
// propagate to the caller — tracing must never affect request handling.
func (c *Client) Send(span Span) {
	go func() {
		body, err := json.Marshal(span)
		if err != nil {
			slog.Error("tracer: marshal span", "error", err)
			return
		}

		req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, c.url+"/spans", bytes.NewReader(body))
		if err != nil {
			slog.Error("tracer: build request", "error", err)
			return
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Service-Key", c.serviceKey)

		resp, err := c.http.Do(req)
		if err != nil {
			slog.Error("tracer: send span", "error", err)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusCreated {
			slog.Error("tracer: unexpected status", "status", resp.StatusCode)
		}
	}()
}

func (c *Client) SendSpan(traceId, op string, start time.Time, end time.Time, count ...int) {
	if c == nil || traceId == "" {
		return
	}
	span := Span{
		TraceID:   traceId,
		SpanID:    uuid.NewString(),
		Service:   c.service,
		Operation: op,
		Status:    "ok",
		StartTime: start,
		EndTime:   end,
	}
	if len(count) > 0 {
		span.Metadata = map[string]any{"count": count[0]}
	}
	c.Send(span)
}

func (c *Client) SendErrorSpan(traceId, op, errMsg string, start, end time.Time) {
	if c == nil || traceId == "" {
		return
	}
	span := Span{
		TraceID:   traceId,
		SpanID:    uuid.NewString(),
		Service:   c.service,
		Operation: op,
		Status:    "error",
		Error:     &errMsg,
		StartTime: start,
		EndTime:   end,
	}
	c.Send(span)
}
