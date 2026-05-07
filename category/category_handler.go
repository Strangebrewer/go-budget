package category

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/Strangebrewer/go-budget/middleware"
	"github.com/Strangebrewer/go-budget/tracer"
)

type Handler struct {
	store  *Store
	tracer *tracer.Client
}

func NewHandler(store *Store, tc *tracer.Client) *Handler {
	return &Handler{store: store, tracer: tc}
}

func (h *Handler) GetAll(w http.ResponseWriter, r *http.Request) {
	userID, err := userIDFromRequest(r)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	traceID := r.Header.Get("X-Trace-ID")
	start := time.Now()

	rows, err := h.store.GetAll(r.Context(), userID)
	if err != nil {
		end := time.Now()
		slog.Error("get all categories", "error", err)
		h.tracer.SendErrorSpan(traceID, "get_all_categories", "internal server error", start, end)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(rows)
	end := time.Now()
	h.tracer.SendSpan(traceID, "get_all_categories", start, end)
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	userID, err := userIDFromRequest(r)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	traceID := r.Header.Get("X-Trace-ID")
	start := time.Now()

	var req CreateCategoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		end := time.Now()
		errMsg := "invalid request body"
		h.tracer.SendErrorSpan(traceID, "create_category", errMsg, start, end)
		http.Error(w, errMsg, http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	if req.Name == "" {
		end := time.Now()
		errMsg := "name is required"
		h.tracer.SendErrorSpan(traceID, "create_category", errMsg, start, end)
		http.Error(w, errMsg, http.StatusBadRequest)
		return
	}

	c, err := h.store.Create(r.Context(), userID, req)
	if err != nil {
		end := time.Now()
		slog.Error("create category", "error", err)
		h.tracer.SendErrorSpan(traceID, "create_category", "internal server error", start, end)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(c)
	end := time.Now()
	h.tracer.SendSpan(traceID, "create_category", start, end)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	userID, err := userIDFromRequest(r)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	traceID := r.Header.Get("X-Trace-ID")
	op := fmt.Sprintf("update_category by id: %s", id)
	start := time.Now()

	var req UpdateCategoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		end := time.Now()
		errMsg := "invalid request body"
		h.tracer.SendErrorSpan(traceID, op, errMsg, start, end)
		http.Error(w, errMsg, http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	c, err := h.store.Update(r.Context(), id, userID, req)
	if err != nil {
		end := time.Now()
		if errors.Is(err, ErrNotFound) {
			h.tracer.SendErrorSpan(traceID, op, "not found", start, end)
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		slog.Error("update category", "id", id, "error", err)
		h.tracer.SendErrorSpan(traceID, op, "internal server error", start, end)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(c)
	end := time.Now()
	h.tracer.SendSpan(traceID, op, start, end)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	userID, err := userIDFromRequest(r)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	traceID := r.Header.Get("X-Trace-ID")
	op := fmt.Sprintf("delete_category by id: %s", id)
	start := time.Now()

	if err := h.store.Delete(r.Context(), id, userID); err != nil {
		end := time.Now()
		slog.Error("delete category", "id", id, "error", err)
		h.tracer.SendErrorSpan(traceID, op, "internal server error", start, end)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
	end := time.Now()
	h.tracer.SendSpan(traceID, op, start, end)
}

func userIDFromRequest(r *http.Request) (uuid.UUID, error) {
	idStr, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		return uuid.UUID{}, errors.New("no user id in context")
	}
	return uuid.Parse(idStr)
}
