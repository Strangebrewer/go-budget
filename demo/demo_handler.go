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

	if err := h.seedIncome(ctx, userID, accountIDs, payload.ExpiresAt); err != nil {
		slog.Error("demo-registered: seed income", "userId", userID, "error", err)
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

// accountIDs index: 0=My Checking, 1=My Savings, 2=My Visa, 3=Her Checking, 4=Her Savings, 5=Her Visa
func (h *Handler) seedAccounts(ctx context.Context, userID uuid.UUID, expiresAt time.Time) ([6]string, error) {
	type seed struct {
		name    string
		balance int32
		accType account.AccountType
		owner   string
	}
	seeds := [6]seed{
		{"My Checking", 324100, account.AccountTypeAsset, "mine"},
		{"My Savings", 1875000, account.AccountTypeAsset, "mine"},
		{"My Visa", 182300, account.AccountTypeDebt, "mine"},
		{"Her Checking", 215600, account.AccountTypeAsset, "hers"},
		{"Her Savings", 2430000, account.AccountTypeAsset, "hers"},
		{"Her Visa", 89200, account.AccountTypeDebt, "hers"},
	}
	var ids [6]string
	for i, s := range seeds {
		a, err := h.accountStore.Create(ctx, userID, account.CreateAccountRequest{
			Name:    s.name,
			Balance: s.balance,
			Type:    s.accType,
			Owner:   s.owner,
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
	owner      string
}

// seedBills creates 8 bills — 4 mine (My Checking), 4 hers (Her Checking). categoryIDs: 2=Other.
func (h *Handler) seedBills(ctx context.Context, userID uuid.UUID, accountIDs [6]string, categoryIDs [3]string, expiresAt time.Time) ([]seededBill, error) {
	type seed struct {
		name      string
		sourceIdx int
		amount    int32
		owner     string
	}
	seeds := []seed{
		{"Mortgage", 0, 155000, "mine"},
		{"Electric", 0, 11800, "mine"},
		{"Natural Gas", 0, 7400, "mine"},
		{"Internet", 0, 8900, "mine"},
		{"Car Insurance", 3, 14200, "hers"},
		{"Cell Phone", 3, 15500, "hers"},
		{"Student Loan", 3, 43000, "hers"},
		{"Water", 3, 6500, "hers"},
	}

	result := make([]seededBill, 0, len(seeds))
	for _, s := range seeds {
		b, err := h.billStore.Create(ctx, userID, bill.CreateBillRequest{
			Name:       s.name,
			SourceID:   accountIDs[s.sourceIdx],
			CategoryID: categoryIDs[2],
			Owner:      s.owner,
		}, &expiresAt)
		if err != nil {
			return nil, fmt.Errorf("create bill %q: %w", s.name, err)
		}
		result = append(result, seededBill{
			id:         b.ID,
			sourceID:   accountIDs[s.sourceIdx],
			categoryID: categoryIDs[2],
			amount:     s.amount,
			owner:      s.owner,
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
				Owner:       b.owner,
				Shared:      true,
			}, &expiresAt)
			if err != nil {
				return fmt.Errorf("create bill payment (bill=%s month=%s): %w", b.id, month, err)
			}
		}
	}
	return nil
}

func (h *Handler) seedTransactions(ctx context.Context, userID uuid.UUID, accountIDs [6]string, categoryIDs [3]string, expiresAt time.Time) error {
	// accountIDs: 0=My Checking, 2=My Visa, 3=Her Checking, 5=Her Visa
	// categoryIDs: 0=Food, 1=Gas, 2=Other
	myChecking := accountIDs[0]
	myVisa := accountIDs[2]
	herChecking := accountIDs[3]
	herVisa := accountIDs[5]
	food := categoryIDs[0]
	gas := categoryIDs[1]
	other := categoryIDs[2]

	months := recentMonths(3)
	m0, m1, m2 := months[0], months[1], months[2]

	type seed struct {
		desc     string
		category string
		amount   int32
		source   string
		owner    string
		month    string
	}
	seeds := []seed{
		{"Whole Foods", food, 8432, myChecking, "mine", m0},
		{"Trader Joe's", food, 6218, herChecking, "hers", m0},
		{"Chipotle", food, 1245, myVisa, "mine", m0},
		{"Sushi restaurant", food, 8900, herVisa, "hers", m0},
		{"Grocery run", food, 7823, myChecking, "mine", m0},
		{"Coffee shop", food, 645, herVisa, "hers", m0},
		{"Pizza delivery", food, 2840, myVisa, "mine", m0},
		{"Shell", gas, 5200, myChecking, "mine", m1},
		{"BP", gas, 4800, herChecking, "hers", m1},
		{"Chevron", gas, 5500, myChecking, "mine", m1},
		{"Car wash", gas, 1600, myChecking, "mine", m1},
		{"Shell", gas, 4900, herChecking, "hers", m1},
		{"BP", gas, 5100, myChecking, "mine", m1},
		{"Amazon", other, 4200, myVisa, "mine", m2},
		{"Target", other, 8900, herVisa, "hers", m2},
		{"Home Depot", other, 12300, myChecking, "mine", m2},
		{"Pharmacy", other, 2345, herChecking, "hers", m2},
		{"Gym membership", other, 4500, myChecking, "mine", m2},
		{"Phone bill", other, 8000, myChecking, "mine", m2},
		{"Farmers market", food, 4500, herChecking, "hers", m1},
	}

	for _, s := range seeds {
		_, err := h.transactionStore.Create(ctx, userID, transaction.CreateTransactionRequest{
			SourceID:    s.source,
			CategoryID:  s.category,
			Amount:      s.amount,
			Month:       s.month,
			Description: s.desc,
			Type:        transaction.TransactionTypeDebit,
			Owner:       s.owner,
			Shared:      true,
		}, &expiresAt)
		if err != nil {
			return fmt.Errorf("create transaction %q: %w", s.desc, err)
		}
	}
	return nil
}

func (h *Handler) seedIncome(ctx context.Context, userID uuid.UUID, accountIDs [6]string, expiresAt time.Time) error {
	months := recentMonths(6)
	myChecking := accountIDs[0]
	herChecking := accountIDs[3]

	type incomeSeed struct {
		amounts []int32
		month   string
	}

	// Base ~$2,875/paycheck; current month gets 1, prior months get 2 each.
	mySeeds := []incomeSeed{
		{[]int32{288000}, months[0]},
		{[]int32{290200, 285600}, months[1]},
		{[]int32{286800, 289400}, months[2]},
		{[]int32{291000, 287300}, months[3]},
		{[]int32{288500, 285900}, months[4]},
		{[]int32{286200, 290100}, months[5]},
	}

	// Base ~$2,310/paycheck.
	herSeeds := []incomeSeed{
		{[]int32{232000}, months[0]},
		{[]int32{231700, 230500}, months[1]},
		{[]int32{229400, 233000}, months[2]},
		{[]int32{232800, 230100}, months[3]},
		{[]int32{231200, 228900}, months[4]},
		{[]int32{229800, 232500}, months[5]},
	}

	for _, s := range mySeeds {
		for _, amount := range s.amounts {
			_, err := h.transactionStore.Create(ctx, userID, transaction.CreateTransactionRequest{
				SourceID:    myChecking,
				Amount:      amount,
				Month:       s.month,
				Description: "Direct Deposit",
				Income:      true,
				Owner:       "mine",
				Shared:      true,
				Type:        transaction.TransactionTypeCredit,
			}, &expiresAt)
			if err != nil {
				return fmt.Errorf("create income (mine month=%s): %w", s.month, err)
			}
		}
	}

	for _, s := range herSeeds {
		for _, amount := range s.amounts {
			_, err := h.transactionStore.Create(ctx, userID, transaction.CreateTransactionRequest{
				SourceID:    herChecking,
				Amount:      amount,
				Month:       s.month,
				Description: "Direct Deposit",
				Income:      true,
				Owner:       "hers",
				Shared:      true,
				Type:        transaction.TransactionTypeCredit,
			}, &expiresAt)
			if err != nil {
				return fmt.Errorf("create income (hers month=%s): %w", s.month, err)
			}
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
