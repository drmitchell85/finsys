package bank

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/drmitchell85/finsys/internal/utils"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"golang.org/x/exp/rand"
)

type BankService interface {
	HasSufficientFunds(ctx context.Context, accountID uuid.UUID, amount decimal.Decimal) (bool, error)
	ReserveFunds(ctx context.Context, accountID uuid.UUID, amount decimal.Decimal) (uuid.UUID, error)
	ReleaseFunds(ctx context.Context, accountID uuid.UUID, reservationID uuid.UUID) error
}

type mockBankService struct {
	db *sql.DB
}

func NewBankService(db *sql.DB) BankService {
	return &mockBankService{
		db: db,
	}
}

func (m *mockBankService) HasSufficientFunds(ctx context.Context, accountID uuid.UUID, amount decimal.Decimal) (bool, error) {
	// network latency sim
	time.Sleep(time.Duration(rand.Intn(50)) * time.Millisecond)

	var availableBalance decimal.Decimal
	var status string

	err := m.db.QueryRowContext(ctx, `
        SELECT 
            ma.balance - COALESCE(SUM(mr.amount), 0) as available_balance,
            ma.status
        FROM mock_accounts ma
        LEFT JOIN mock_reservations mr ON ma.id = mr.account_id 
            AND mr.expires_at > NOW()
        WHERE ma.id = $1
        GROUP BY ma.id, ma.balance, ma.status
    `, accountID).Scan(&availableBalance, &status)

	if err == sql.ErrNoRows {
		return false, utils.NewNotFoundError(fmt.Sprintf("account not found"), err)
	}
	if status != "active" {
		return false, utils.NewForbiddenError(fmt.Sprintf("account %s", status), err)
	}

	return availableBalance.GreaterThanOrEqual(amount), nil
}

func (m *mockBankService) ReserveFunds(ctx context.Context, accountID uuid.UUID, amount decimal.Decimal) (uuid.UUID, error) {
	tx, err := m.db.BeginTx(ctx, nil)
	if err != nil {
		return uuid.Nil, utils.NewInternalError(err)
	}
	defer tx.Rollback()

	// check + reserve atomically
	var currentBalance decimal.Decimal
	err = tx.QueryRowContext(ctx,
		"SELECT balance FROM mock_accounts WHERE id = $1 AND status = 'active' FOR UPDATE",
		accountID).Scan(&currentBalance)

	if err != nil {
		return uuid.Nil, utils.NewInternalError(err)
	}

	if currentBalance.LessThan(amount) {
		return uuid.Nil, utils.NewForbiddenError("insufficient funds", fmt.Errorf("error"))
	}

	// create reservation
	var reservationID uuid.UUID
	err = tx.QueryRowContext(ctx,
		"INSERT INTO mock_reservations (account_id, amount, expires_at) VALUES ($1, $2, $3) RETURNING id",
		accountID, amount, time.Now().Add(time.Hour)).Scan(&reservationID)

	if err != nil {
		return uuid.Nil, utils.NewInternalError(err)
	}

	return reservationID, tx.Commit()
}

func (m *mockBankService) ReleaseFunds(ctx context.Context, accountID uuid.UUID, reservationID uuid.UUID) error {
	return nil
}
