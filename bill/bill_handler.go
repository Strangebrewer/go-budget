package bill

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/Strangebrewer/go-budget/middleware"
	"github.com/Strangebrewer/go-budget/transaction"
)

type Handler struct {
	store            *Store
	transactionStore *transaction.Store
}

func NewHandler(store *Store, transactionStore *transaction.Store) *Handler {
	return &Handler{store: store, transactionStore: transactionStore}
}

// GetAll returns all active bills, each with transactions from the current month
// and two preceding months when a ?month=YYYY-MM query param is provided.
func (h *Handler) GetAll(w http.ResponseWriter, r *http.Request) {
	userID, err := userIDFromRequest(r)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	month := r.URL.Query().Get("month")
	if month != "" {
		if _, err := time.Parse("2006-01", month); err != nil {
			http.Error(w, "invalid month format, expected YYYY-MM", http.StatusBadRequest)
			return
		}
	}

	bills, err := h.store.GetAll(r.Context(), userID)
	if err != nil {
		slog.Error("get all bills", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	var txns []transaction.Transaction
	if month != "" {
		rows, err := h.transactionStore.GetByMonth(r.Context(), userID, month)
		if err != nil {
			slog.Error("get transactions for bills", "month", month, "error", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		txns = rows
	}

	resp := make([]BillResponse, len(bills))
	for i, b := range bills {
		resp[i].Bill = b
		if month != "" {
			var billTxns []transaction.Transaction
			for _, t := range txns {
				if t.BillID != nil && *t.BillID == b.ID {
					billTxns = append(billTxns, t)
				}
			}
			resp[i].Transactions = billTxns
		}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	userID, err := userIDFromRequest(r)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	
	var req CreateBillRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	if req.Name == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}
	if req.SourceID == "" {
		http.Error(w, "sourceId is required", http.StatusBadRequest)
		return
	}

	if middleware.IsDemoFromContext(r.Context()) {
		count, err := h.store.CountByUser(r.Context(), userID)
		if err != nil {
			slog.Error("count bills", "error", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		if count >= 12 {
			http.Error(w, "demo bill limit reached", http.StatusForbidden)
			return
		}
	}

	b, err := h.store.Create(r.Context(), userID, req, nil)
	if err != nil {
		slog.Error("create bill", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(b)
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
	
	var req UpdateBillRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	b, err := h.store.Update(r.Context(), id, userID, req)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		slog.Error("update bill", "id", id, "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(b)
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
	
	if err := h.store.Delete(r.Context(), id, userID); err != nil {
		slog.Error("delete bill", "id", id, "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// PayBill creates a transaction recording payment of the bill for a given month.
func (h *Handler) PayBill(w http.ResponseWriter, r *http.Request) {
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
	
	b, err := h.store.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		slog.Error("pay bill: get bill", "id", id, "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	var req PayBillRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	if _, err := time.Parse("2006-01", req.Month); err != nil {
		http.Error(w, "invalid month format, expected YYYY-MM", http.StatusBadRequest)
		return
	}

	// Use the bill's source account unless the request overrides it.
	sourceID := b.SourceID
	if req.SourceID != "" {
		sourceID = req.SourceID
	}

	if middleware.IsDemoFromContext(r.Context()) {
		count, err := h.transactionStore.CountByBill(r.Context(), id)
		if err != nil {
			slog.Error("count bill payments", "bill_id", id, "error", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		if count >= 20 {
			http.Error(w, "demo bill payment limit reached", http.StatusForbidden)
			return
		}
	}

	txnReq := transaction.CreateTransactionRequest{
		SourceID:    sourceID,
		BillID:      b.ID,
		CategoryID:  uuidPtrToStr(b.CategoryID),
		Amount:      req.Amount,
		Month:       req.Month,
		Description: req.Description,
		Income:      false,
		Owner:       b.Owner,
		Shared:      b.Shared,
		Type:        transaction.TransactionTypeDebit,
	}

	t, err := h.transactionStore.CreateFromBill(r.Context(), userID, txnReq, nil)
	if err != nil {
		slog.Error("pay bill: create transaction", "bill_id", id, "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(t)
}

func userIDFromRequest(r *http.Request) (uuid.UUID, error) {
	idStr, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		return uuid.UUID{}, errors.New("no user id in context")
	}
	return uuid.Parse(idStr)
}

func uuidPtrToStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
