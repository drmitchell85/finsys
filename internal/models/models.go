package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type TransactionStatus string

const (
	TransactionPending    TransactionStatus = "pending"
	TransactionProcessing TransactionStatus = "processing"
	TransactionCompleted  TransactionStatus = "completed"
	TransactionFailed     TransactionStatus = "failed"
)

type Message struct {
	Type      string          `json:"type"`      // "transaction", "notification", etc
	Payload   json.RawMessage `json:"payload"`   // type-specific payload
	Attempts  int             `json:"attempts"`  // for tracking retries
	Timestamp int64           `json:"timestamp"` // unix timestamp when created
}

// Type-specific payloads
type TransactionPayload struct {
	TransactionID  uuid.UUID `json:"transaction_id"`
	IdempotencyKey string    `json:"idempotency_key"`
	Operation      string    `json:"operation"` // "process", "refund", etc.
}

type NotificationPayload struct {
	UserID      uuid.UUID `json:"user_id"`
	TemplateID  string    `json:"template_id"`
	Destination string    `json:"destination"` // email, phone, etc.
	Data        any       `json:"data"`        // template data
}

type Transaction struct {
	ID             uuid.UUID         `json:"id,omitempty"`
	IdempotencyKey string            `json:"idempotency_key" validate:"required"`
	FromAccountID  uuid.UUID         `json:"from_account_id" validate:"required"`
	ToAccountID    *uuid.UUID        `json:"to_account_id"`
	Amount         decimal.Decimal   `json:"amount" validate:"required,gt=0"`
	Currency       string            `json:"currency" validate:"required,len=3"`
	Status         TransactionStatus `json:"status"`
	CreatedAt      time.Time         `json:"created_at,omitempty"` // add these
	UpdatedAt      time.Time         `json:"updated_at,omitempty"`
	ReservationID  uuid.UUID         `json:"bank_reservation_id" validate:"required"`
}

type IdempotencyCache struct {
	TransactionID uuid.UUID         `json:"transaction_id"`
	Status        TransactionStatus `json:"status"`
	Response      json.RawMessage   `json:"response"` // the actual API response
	CreatedAt     time.Time         `json:"created_at"`
	CompletedAt   *time.Time        `json:"completed_at,omitempty"`
	ErrorMessage  string            `json:"error_message,omitempty"`
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
	TransactionID uuid.UUID         `json:"transaction_id"`
	Status        TransactionStatus `json:"status"`
	CreatedAt     time.Time         `json:"created_at"`
}
