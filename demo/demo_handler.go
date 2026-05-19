package demo

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/Strangebrewer/go-budget/account"
	"github.com/Strangebrewer/go-budget/bill"
	"github.com/Strangebrewer/go-budget/category"
	"github.com/Strangebrewer/go-budget/tracer"
	"github.com/Strangebrewer/go-budget/transaction"
)

type Handler struct {
	accountStore     *account.Store
	billStore        *bill.Store
	categoryStore    *category.Store
	transactionStore *transaction.Store
	tracer           *tracer.Client
}

func NewHandler(
	accountStore *account.Store,
	billStore *bill.Store,
	categoryStore *category.Store,
	transactionStore *transaction.Store,
	tc *tracer.Client,
) *Handler {
	return &Handler{
		accountStore:     accountStore,
		billStore:        billStore,
		categoryStore:    categoryStore,
		transactionStore: transactionStore,
		tracer:           tc,
	}
}

func (h *Handler) HandleDemoRegistered(w http.ResponseWriter, r *http.Request) {
	var msg pubSubMessage
	if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
		slog.Error("demo-registered: decode body", "error", err)
		w.WriteHeader(http.StatusOK)
		return
	}

	raw, err := base64.StdEncoding.DecodeString(msg.Message.Data)
	if err != nil {
		slog.Error("demo-registered: decode base64", "error", err)
		w.WriteHeader(http.StatusOK)
		return
	}

	var payload demoRegisteredPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		slog.Error("demo-registered: unmarshal payload", "error", err)
		w.WriteHeader(http.StatusOK)
		return
	}

	userID, err := uuid.Parse(payload.UserID)
	if err != nil {
		slog.Error("demo-registered: parse userID", "error", err)
		w.WriteHeader(http.StatusOK)
		return
	}

	start := time.Now()
	ctx := r.Context()

	categoryIDs, err := h.seedCategories(ctx, userID, payload.ExpiresAt)
	if err != nil {
		slog.Error("demo-registered: seed categories", "userId", userID, "error", err)
		h.sendErrorSpan(payload.TraceID, err, start)
		w.WriteHeader(http.StatusOK)
		return
	}

	accountIDs, err := h.seedAccounts(ctx, userID, payload.ExpiresAt)
	if err != nil {
		slog.Error("demo-registered: seed accounts", "userId", userID, "error", err)
		h.sendErrorSpan(payload.TraceID, err, start)
		w.WriteHeader(http.StatusOK)
		return
	}

	bills, err := h.seedBills(ctx, userID, accountIDs, categoryIDs, payload.ExpiresAt)
	if err != nil {
		slog.Error("demo-registered: seed bills", "userId", userID, "error", err)
		h.sendErrorSpan(payload.TraceID, err, start)
		w.WriteHeader(http.StatusOK)
		return
	}

	if err := h.seedBillPayments(ctx, userID, bills, payload.ExpiresAt); err != nil {
		slog.Error("demo-registered: seed bill payments", "userId", userID, "error", err)
		h.sendErrorSpan(payload.TraceID, err, start)
		w.WriteHeader(http.StatusOK)
		return
	}

	if err := h.seedTransactions(ctx, userID, accountIDs, categoryIDs, payload.ExpiresAt); err != nil {
		slog.Error("demo-registered: seed transactions", "userId", userID, "error", err)
		h.sendErrorSpan(payload.TraceID, err, start)
		w.WriteHeader(http.StatusOK)
		return
	}

	if h.tracer != nil && payload.TraceID != "" {
		h.tracer.SendSpan(payload.TraceID, "demo seed", start, time.Now())
	}

	w.WriteHeader(http.StatusOK)
}

// categoryIDs index: 0=Food, 1=Gas, 2=Other
func (h *Handler) seedCategories(ctx context.Context, userID uuid.UUID, expiresAt time.Time) ([3]string, error) {
	names := [3]string{"Food", "Gas", "Other"}
	var ids [3]string
	for i, name := range names {
		c, err := h.categoryStore.Create(ctx, userID, category.CreateCategoryRequest{Name: name}, &expiresAt)
		if err != nil {
			return ids, fmt.Errorf("create category %q: %w", name, err)
		}
		ids[i] = c.ID
	}
	return ids, nil
}

// accountIDs index: 0=Checking, 1=Savings, 2=Visa
func (h *Handler) seedAccounts(ctx context.Context, userID uuid.UUID, expiresAt time.Time) ([3]string, error) {
	type seed struct {
		name    string
		balance int32
		accType account.AccountType
	}
	seeds := [3]seed{
		{"Checking", 235400, account.AccountTypeAsset},
		{"Savings", 1250000, account.AccountTypeAsset},
		{"Visa", 142300, account.AccountTypeDebt},
	}
	var ids [3]string
	for i, s := range seeds {
		a, err := h.accountStore.Create(ctx, userID, account.CreateAccountRequest{
			Name:    s.name,
			Balance: s.balance,
			Type:    s.accType,
		}, &expiresAt)
		if err != nil {
			return ids, fmt.Errorf("create account %q: %w", s.name, err)
		}
		ids[i] = a.ID
	}
	return ids, nil
}

type seededBill struct {
	id         string
	sourceID   string
	categoryID string
	amount     int32
}

// seedBills creates 4 bills. accountIDs: 0=Checking, 1=Savings, 2=Visa. categoryIDs: 0=Food, 1=Gas, 2=Other.
func (h *Handler) seedBills(ctx context.Context, userID uuid.UUID, accountIDs [3]string, categoryIDs [3]string, expiresAt time.Time) ([]seededBill, error) {
	type seed struct {
		name       string
		sourceIdx  int
		categoryID string
		dueDay     int32
		amount     int32
	}
	seeds := []seed{
		{"Netflix", 2, categoryIDs[2], 15, 1999},
		{"Spotify", 2, categoryIDs[2], 1, 1099},
		{"Electric", 0, categoryIDs[2], 10, 8500},
		{"Car Insurance", 0, categoryIDs[2], 20, 14200},
	}

	result := make([]seededBill, 0, len(seeds))
	for _, s := range seeds {
		b, err := h.billStore.Create(ctx, userID, bill.CreateBillRequest{
			Name:       s.name,
			SourceID:   accountIDs[s.sourceIdx],
			CategoryID: s.categoryID,
			DueDay:     s.dueDay,
		}, &expiresAt)
		if err != nil {
			return nil, fmt.Errorf("create bill %q: %w", s.name, err)
		}
		result = append(result, seededBill{
			id:         b.ID,
			sourceID:   accountIDs[s.sourceIdx],
			categoryID: s.categoryID,
			amount:     s.amount,
		})
	}
	return result, nil
}

func (h *Handler) seedBillPayments(ctx context.Context, userID uuid.UUID, bills []seededBill, expiresAt time.Time) error {
	months := recentMonths(6)
	for _, b := range bills {
		for _, month := range months {
			_, err := h.transactionStore.CreateFromBill(ctx, userID, transaction.CreateTransactionRequest{
				SourceID:    b.sourceID,
				BillID:      b.id,
				CategoryID:  b.categoryID,
				Amount:      b.amount,
				Month:       month,
				Description: "",
				Type:        transaction.TransactionTypeDebit,
			}, &expiresAt)
			if err != nil {
				return fmt.Errorf("create bill payment (bill=%s month=%s): %w", b.id, month, err)
			}
		}
	}
	return nil
}

func (h *Handler) seedTransactions(ctx context.Context, userID uuid.UUID, accountIDs [3]string, categoryIDs [3]string, expiresAt time.Time) error {
	// 0=Checking, 1=Savings, 2=Visa  |  categoryIDs: 0=Food, 1=Gas, 2=Other
	checking := accountIDs[0]
	visa := accountIDs[2]
	food := categoryIDs[0]
	gas := categoryIDs[1]
	other := categoryIDs[2]

	months := recentMonths(3)
	m0, m1, m2 := months[0], months[1], months[2]

	type seed struct {
		desc       string
		category   string
		amount     int32
		source     string
		month      string
	}
	seeds := []seed{
		{"Whole Foods", food, 8432, checking, m0},
		{"Trader Joe's", food, 6218, checking, m0},
		{"Chipotle", food, 1245, visa, m0},
		{"Sushi restaurant", food, 8900, visa, m0},
		{"Grocery run", food, 7823, checking, m0},
		{"Coffee shop", food, 645, visa, m0},
		{"Pizza delivery", food, 2840, visa, m0},
		{"Shell", gas, 5200, checking, m1},
		{"BP", gas, 4800, checking, m1},
		{"Chevron", gas, 5500, checking, m1},
		{"Car wash", gas, 1600, checking, m1},
		{"Shell", gas, 4900, checking, m1},
		{"BP", gas, 5100, checking, m1},
		{"Amazon", other, 4200, visa, m2},
		{"Target", other, 8900, visa, m2},
		{"Home Depot", other, 12300, checking, m2},
		{"Pharmacy", other, 2345, checking, m2},
		{"Gym membership", other, 4500, checking, m2},
		{"Phone bill", other, 8000, checking, m2},
		{"Farmers market", food, 4500, checking, m1},
	}

	for _, s := range seeds {
		_, err := h.transactionStore.Create(ctx, userID, transaction.CreateTransactionRequest{
			SourceID:    s.source,
			CategoryID:  s.category,
			Amount:      s.amount,
			Month:       s.month,
			Description: s.desc,
			Type:        transaction.TransactionTypeDebit,
		}, &expiresAt)
		if err != nil {
			return fmt.Errorf("create transaction %q: %w", s.desc, err)
		}
	}
	return nil
}

func (h *Handler) sendErrorSpan(traceID string, err error, start time.Time) {
	if h.tracer != nil && traceID != "" {
		h.tracer.SendErrorSpan(traceID, "demo seed", err.Error(), start, time.Now())
	}
}

// recentMonths returns n month strings (YYYY-MM) ending with the current month, newest first.
func recentMonths(n int) []string {
	now := time.Now()
	months := make([]string, n)
	y, m, _ := now.Date()
	for i := 0; i < n; i++ {
		months[i] = fmt.Sprintf("%d-%02d", y, int(m))
		m--
		if m == 0 {
			m = 12
			y--
		}
	}
	return months
}
