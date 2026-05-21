package rube

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"math/rand"
	"net/http"
	"time"

	"github.com/Strangebrewer/go-budget/tracer"
	"github.com/google/uuid"
)

var words = []string{
	"isomer", "welfare", "thermal", "flux", "unemployment",
	"salinity", "radiation", "democracy", "tectonic", "entropy",
	"corruption", "volcanic", "catalyst", "education", "glacial",
	"polymer", "sanitation", "seismic", "isotope", "healthcare",
	"drought", "plasma", "nutrition", "deforestation", "conflict",
}

type ChainRequest struct {
	UserId    string     `json:"userId"`
	Words     []string   `json:"words"`
	ExpiresAt *time.Time `json:"expiresAt,omitempty"`
}

type Handler struct {
	nextURL string
	tracer  *tracer.Client
	client  *http.Client
}

func NewHandler(nextURL string, tc *tracer.Client) *Handler {
	return &Handler{
		nextURL: nextURL,
		tracer:  tc,
		client:  &http.Client{Timeout: 15 * time.Second},
	}
}

func (h *Handler) Chain(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	var req ChainRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	traceID := r.Header.Get("X-Trace-ID")
	word := words[rand.Intn(len(words))]
	req.Words = append(req.Words, word)

	body, err := json.Marshal(req)
	if err != nil {
		slog.Error("rube marshal", "error", err)
		if h.tracer != nil && traceID != "" {
			h.tracer.SendErrorSpan(traceID, "POST /rube", "rube marshal error", start, time.Now())
		}
		http.Error(w, "rube marshal error", http.StatusInternalServerError)
		return
	}

	downstream, err := http.NewRequest(http.MethodPost, h.nextURL, bytes.NewReader(body))
	if err != nil {
		slog.Error("rube build request", "error", err)
		if h.tracer != nil && traceID != "" {
			h.tracer.SendErrorSpan(traceID, "POST /rube", "rube build request error", start, time.Now())
		}
		http.Error(w, "rube build request error", http.StatusInternalServerError)
		return
	}
	downstream.Header.Set("Content-Type", "application/json")
	downstream.Header.Set("X-Trace-ID", traceID)

	resp, err := h.client.Do(downstream)
	if err != nil {
		slog.Error("rube call next", "url", h.nextURL, "error", err)
		if h.tracer != nil && traceID != "" {
			h.tracer.SendErrorSpan(traceID, "POST /rube", "rube call next error", start, time.Now())
		}
		http.Error(w, "rube call next error", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	if h.tracer != nil && traceID != "" {
		h.tracer.Send(tracer.Span{
			TraceID:   traceID,
			SpanID:    uuid.NewString(),
			Service:   "go-budget",
			Operation: "POST /rube",
			Status:    "ok",
			StartTime: start,
			EndTime:   time.Now(),
			Metadata:  map[string]any{"words": req.Words},
		})
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}
