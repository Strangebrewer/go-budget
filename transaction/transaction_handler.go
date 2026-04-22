package transaction

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/Strangebrewer/go-budget/middleware"
)

type Handler struct {
	store *Store
}

func NewHandler(store *Store) *Handler {
	return &Handler{store: store}
}

func (h *Handler) GetAll(w http.ResponseWriter, r *http.Request) {
	userID, err := userIDFromRequest(r)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

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

	switch {
	case income && hasMonth:
		txns, err := h.store.GetIncomeByMonth(r.Context(), userID, month)
		if err != nil {
			slog.Error("get income transactions by month", "month", month, "error", err)
			http.Error(w, "server error", http.StatusInternalServerError)
			return
		}
		_ = json.NewEncoder(w).Encode(txns)
	case income:
		txns, err := h.store.GetIncome(r.Context(), userID)
		if err != nil {
			slog.Error("get income transactions", "error", err)
			http.Error(w, "server error", http.StatusInternalServerError)
			return
		}
		_ = json.NewEncoder(w).Encode(txns)
	case hasMonth && hasCategories:
		txns, err := h.store.GetByMonthAndCategories(r.Context(), userID, month, categoryIDs)
		if err != nil {
			slog.Error("get transactions by month and categories", "month", month, "error", err)
			http.Error(w, "server error", http.StatusInternalServerError)
			return
		}
		_ = json.NewEncoder(w).Encode(txns)
	case hasMonth:
		txns, err := h.store.GetByBillMonths(r.Context(), userID, month)
		if err != nil {
			slog.Error("get transactions by month", "month", month, "error", err)
			http.Error(w, "server error", http.StatusInternalServerError)
			return
		}
		_ = json.NewEncoder(w).Encode(txns)
	case hasCategories:
		txns, err := h.store.GetByCategories(r.Context(), userID, categoryIDs)
		if err != nil {
			slog.Error("get transactions by categories", "error", err)
			http.Error(w, "server error", http.StatusInternalServerError)
			return
		}
		_ = json.NewEncoder(w).Encode(txns)
	default:
		txns, err := h.store.GetAll(r.Context(), userID)
		if err != nil {
			slog.Error("get all transactions", "error", err)
			http.Error(w, "server error", http.StatusInternalServerError)
			return
		}
		_ = json.NewEncoder(w).Encode(txns)
	}
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	userID, err := userIDFromRequest(r)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var req CreateTransactionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	if req.Date == "" {
		http.Error(w, "date is required", http.StatusBadRequest)
		return
	}

	t, err := h.store.Create(r.Context(), userID, req)
	if err != nil {
		slog.Error("create transaction", "error", err)
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(t)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	var req UpdateTransactionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	t, err := h.store.Update(r.Context(), id, req)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		slog.Error("update transaction", "id", id, "error", err)
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(t)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	if err := h.store.Delete(r.Context(), id); err != nil {
		slog.Error("delete transaction", "id", id, "error", err)
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func userIDFromRequest(r *http.Request) (uuid.UUID, error) {
	idStr, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		return uuid.UUID{}, errors.New("no user id in context")
	}
	return uuid.Parse(idStr)
}
