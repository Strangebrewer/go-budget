package transaction

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
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

	month := r.URL.Query().Get("month")
	categoryParam := r.URL.Query().Get("category")
	income := r.URL.Query().Get("income") == "true"

	if month != "" {
		if _, err := time.Parse("2006-01", month); err != nil {
			http.Error(w, "invalid month format, expected YYYY-MM", http.StatusBadRequest)
			return
		}
	}

	var categoryIDs []uuid.UUID
	if categoryParam != "" {
		for _, raw := range strings.Split(categoryParam, ",") {
			id, err := uuid.Parse(strings.TrimSpace(raw))
			if err != nil {
				http.Error(w, "invalid category id: "+raw, http.StatusBadRequest)
				return
			}
			categoryIDs = append(categoryIDs, id)
		}
	}

	if income && len(categoryIDs) > 0 {
		http.Error(w, "income and category filters cannot be combined", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	hasMonth := month != ""
	hasCategories := len(categoryIDs) > 0

	start := time.Now()

	onFail := func(msg string, logArgs ...any) {
		errMsg := "internal server error"
		end := time.Now()
		if len(logArgs) > 0 {
			slog.Error(msg, logArgs...)
		}
		h.tracer.SendErrorSpan(traceID, "get_all_transactions", errMsg, start, end)
		http.Error(w, errMsg, http.StatusInternalServerError)
	}

	switch {
	case income && hasMonth:
		txns, err := h.store.GetIncomeByMonth(r.Context(), userID, month)
		if err != nil {
			onFail("get income transactions by month", "month", month, "error", err)
			return
		}
		_ = json.NewEncoder(w).Encode(txns)
	case income:
		txns, err := h.store.GetIncome(r.Context(), userID)
		if err != nil {
			onFail("get income transactions", "error", err)
			return
		}
		_ = json.NewEncoder(w).Encode(txns)
	case hasMonth && hasCategories:
		txns, err := h.store.GetByMonthAndCategories(r.Context(), userID, month, categoryIDs)
		if err != nil {
			onFail("get transactions by month and categories", "month", month, "error", err)
			return
		}
		_ = json.NewEncoder(w).Encode(txns)
	case hasMonth:
		txns, err := h.store.GetByMonth(r.Context(), userID, month)
		if err != nil {
			onFail("get transactions by month", "month", month, "error", err)
			return
		}
		_ = json.NewEncoder(w).Encode(txns)
	case hasCategories:
		txns, err := h.store.GetByCategories(r.Context(), userID, categoryIDs)
		if err != nil {
			onFail("get transactions by categories", "error", err)
			return
		}
		_ = json.NewEncoder(w).Encode(txns)
	default:
		txns, err := h.store.GetAll(r.Context(), userID)
		if err != nil {
			onFail("get all transactions", "error", err)
			return
		}
		_ = json.NewEncoder(w).Encode(txns)
	}

	end := time.Now()
	h.tracer.SendSpan(traceID, "get_all_transactions", start, end)
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	userID, err := userIDFromRequest(r)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	traceID := r.Header.Get("X-Trace-ID")

	start := time.Now()
	var req CreateTransactionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		end := time.Now()
		errMsg := "invalid request body"
		h.tracer.SendErrorSpan(traceID, "create_transaction", errMsg, start, end)
		http.Error(w, errMsg, http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	if req.Month == "" {
		end := time.Now()
		errMsg := "month is required"
		h.tracer.SendErrorSpan(traceID, "create_transaction", errMsg, start, end)
		http.Error(w, errMsg, http.StatusBadRequest)
		return
	}

	if err := validateType(req.Type); err != nil {
		end := time.Now()
		h.tracer.SendErrorSpan(traceID, "create_transaction", err.Error(), start, end)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	t, err := h.store.Create(r.Context(), userID, req)
	if err != nil {
		end := time.Now()
		slog.Error("create transaction", "error", err)
		h.tracer.SendErrorSpan(traceID, "create_transaction", "internal server error", start, end)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(t)
	end := time.Now()
	h.tracer.SendSpan(traceID, "create_transaction", start, end)
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
	op := fmt.Sprintf("update_transaction by id: %s", id)

	start := time.Now()
	var req UpdateTransactionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		end := time.Now()
		errMsg := "invalid request body"
		h.tracer.SendErrorSpan(traceID, op, errMsg, start, end)
		http.Error(w, errMsg, http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	if err := validateType(req.Type); err != nil {
		end := time.Now()
		h.tracer.SendErrorSpan(traceID, op, err.Error(), start, end)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	t, err := h.store.Update(r.Context(), id, userID, req)
	if err != nil {
		end := time.Now()
		if errors.Is(err, ErrNotFound) {
			h.tracer.SendErrorSpan(traceID, op, "not found", start, end)
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		slog.Error("update transaction", "id", id, "error", err)
		h.tracer.SendErrorSpan(traceID, op, "internal server error", start, end)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(t)
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
	op := fmt.Sprintf("delete_transaction by id: %s", id)

	start := time.Now()
	if err := h.store.Delete(r.Context(), id, userID); err != nil {
		end := time.Now()
		slog.Error("delete transaction", "id", id, "error", err)
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
