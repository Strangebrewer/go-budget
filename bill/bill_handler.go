package bill

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
	"github.com/Strangebrewer/go-budget/transaction"
)

type Handler struct {
	store            *Store
	transactionStore *transaction.Store
	tracer           *tracer.Client
}

func NewHandler(store *Store, transactionStore *transaction.Store, tc *tracer.Client) *Handler {
	return &Handler{store: store, transactionStore: transactionStore, tracer: tc}
}

// GetAll returns all active bills, each with transactions from the current month
// and two preceding months when a ?month=YYYY-MM query param is provided.
func (h *Handler) GetAll(w http.ResponseWriter, r *http.Request) {
	userID, err := userIDFromRequest(r)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	traceID := r.Header.Get("X-Trace-ID")

	month := r.URL.Query().Get("month")
	if month != "" {
		if _, err := time.Parse("2006-01", month); err != nil {
			http.Error(w, "invalid month format, expected YYYY-MM", http.StatusBadRequest)
			return
		}
	}

	start := time.Now()
	bills, err := h.store.GetAll(r.Context(), userID)
	if err != nil {
		end := time.Now()
		slog.Error("get all bills", "error", err)
		errMsg := "server error"
		h.tracer.SendErrorSpan(traceID, "get_all_bills", errMsg, start, end)
		http.Error(w, errMsg, http.StatusInternalServerError)
		return
	}

	var txns []transaction.Transaction
	if month != "" {
		rows, err := h.transactionStore.GetByMonth(r.Context(), userID, month)
		if err != nil {
			end := time.Now()
			slog.Error("get transactions for bills", "month", month, "error", err)
			errMsg := "server error"
			h.tracer.SendErrorSpan(traceID, "get_all_bills", errMsg, start, end)
			http.Error(w, errMsg, http.StatusInternalServerError)
			return
		}
		txns = rows
	}

	end := time.Now()

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

	h.tracer.SendSpan(traceID, "get_all_bills", start, end, len(resp))

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	userID, err := userIDFromRequest(r)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	traceID := r.Header.Get("X-Trace-ID")

	start := time.Now()
	var req CreateBillRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		end := time.Now()
		errMsg := "invalid request body"
		h.tracer.SendErrorSpan(traceID, "create_bill", errMsg, start, end)
		http.Error(w, errMsg, http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	if req.Name == "" {
		end := time.Now()
		errMsg := "name is required"
		h.tracer.SendErrorSpan(traceID, "create_bill", errMsg, start, end)
		http.Error(w, errMsg, http.StatusBadRequest)
		return
	}
	if req.SourceID == "" {
		end := time.Now()
		errMsg := "sourceId is required"
		h.tracer.SendErrorSpan(traceID, "create_bill", errMsg, start, end)
		http.Error(w, errMsg, http.StatusBadRequest)
		return
	}

	b, err := h.store.Create(r.Context(), userID, req)
	end := time.Now()
	if err != nil {
		slog.Error("create bill", "error", err)
		errMsg := "server error"
		h.tracer.SendErrorSpan(traceID, "create_bill", errMsg, start, end)
		http.Error(w, errMsg, http.StatusInternalServerError)
		return
	}

	h.tracer.SendSpan(traceID, "create_bill", start, end)

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

	traceID := r.Header.Get("X-Trace-ID")
	op := fmt.Sprintf("update_bill by id: %s", id)

	start := time.Now()
	var req UpdateBillRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		end := time.Now()
		errMsg := "invalid request body"
		h.tracer.SendErrorSpan(traceID, op, errMsg, start, end)
		http.Error(w, errMsg, http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	b, err := h.store.Update(r.Context(), id, userID, req)
	end := time.Now()
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			errMsg := "not found"
			h.tracer.SendErrorSpan(traceID, op, errMsg, start, end)
			http.Error(w, errMsg, http.StatusNotFound)
			return
		}
		slog.Error("update bill", "id", id, "error", err)
		errMsg := "server error"
		h.tracer.SendErrorSpan(traceID, op, errMsg, start, end)
		http.Error(w, errMsg, http.StatusInternalServerError)
		return
	}

	h.tracer.SendSpan(traceID, op, start, end)

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

	traceID := r.Header.Get("X-Trace-ID")
	op := fmt.Sprintf("delete_bill by id: %s", id)

	start := time.Now()
	if err := h.store.Delete(r.Context(), id, userID); err != nil {
		end := time.Now()
		slog.Error("delete bill", "id", id, "error", err)
		errMsg := "server error"
		h.tracer.SendErrorSpan(traceID, op, errMsg, start, end)
		http.Error(w, errMsg, http.StatusInternalServerError)
		return
	}

	end := time.Now()
	h.tracer.SendSpan(traceID, op, start, end)

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

	traceID := r.Header.Get("X-Trace-ID")
	op := fmt.Sprintf("pay_bill by id: %s", id)

	start := time.Now()
	b, err := h.store.GetByID(r.Context(), id)
	if err != nil {
		end := time.Now()
		if errors.Is(err, ErrNotFound) {
			h.tracer.SendErrorSpan(traceID, op, "not found", start, end)
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		slog.Error("pay bill: get bill", "id", id, "error", err)
		errMsg := "server error"
		h.tracer.SendErrorSpan(traceID, op, errMsg, start, end)
		http.Error(w, errMsg, http.StatusInternalServerError)
		return
	}

	var req PayBillRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		end := time.Now()
		errMsg := "invalid request body"
		h.tracer.SendErrorSpan(traceID, op, errMsg, start, end)
		http.Error(w, errMsg, http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	if _, err := time.Parse("2006-01", req.Month); err != nil {
		end := time.Now()
		errMsg := "invalid month format, expected YYYY-MM"
		h.tracer.SendErrorSpan(traceID, op, errMsg, start, end)
		http.Error(w, errMsg, http.StatusBadRequest)
		return
	}

	// Use the bill's source account unless the request overrides it.
	sourceID := b.SourceID
	if req.SourceID != "" {
		sourceID = req.SourceID
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

	t, err := h.transactionStore.CreateFromBill(r.Context(), userID, txnReq)
	end := time.Now()
	if err != nil {
		slog.Error("pay bill: create transaction", "bill_id", id, "error", err)
		errMsg := "server error"
		h.tracer.SendErrorSpan(traceID, op, errMsg, start, end)
		http.Error(w, errMsg, http.StatusInternalServerError)
		return
	}

	h.tracer.SendSpan(traceID, op, start, end)

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
