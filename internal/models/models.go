package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type Message struct {
	ID        string          `json:"id"`
	Type      string          `json:"type"`
	Payload   json.RawMessage `json:"payload"`
	Attempts  int             `json:"attempts"`
	Timestamp int64           `json:"timestamp"`
}

type CreateTransactionRequest struct {
	IdempotencyKey string          `json:"idempotency_key" validate:"required"`
	FromAccountID  uuid.UUID       `json:"from_account_id" validate:"required"`
	ToAccountID    *uuid.UUID      `json:"to_account_id" validate:"required"`
	Amount         decimal.Decimal `json:"amount" validate:"required,gt=0"`
	Currency       string          `json:"currency" validate:"required,len=3"`
	Description    string          `json:"description,omitempty"`
	Metadata       map[string]any  `json:"metadata,omitempty"`
}

type CreateTransactionResponse struct {
	TransactionID uuid.UUID `json:"transaction_id"`
	Status        string    `json:"status"` // "pending"
	CreatedAt     time.Time `json:"created_at"`
}
