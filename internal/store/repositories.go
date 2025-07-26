package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/drmitchell85/finsys/internal/models"
	"github.com/drmitchell85/finsys/internal/utils"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type RepositoryService interface {
	AccountExists(accountid uuid.UUID) error
	CheckIdempotencyKey(ctx context.Context, ikey string) (string, error)
	StoreIdempotencyKey(ctx context.Context, key string, data *models.IdempotencyCache, expiration time.Duration) error
	GetIdempotencyCache(ctx context.Context, key string) (*models.IdempotencyCache, error)
	GetTransactionByIdempotencyKey(ctx context.Context, idempKey string) (*models.Transaction, error)
	CreateTransaction(ctx context.Context, tx *models.Transaction) (uuid.UUID, time.Time, error)
	GetExternalBankAccountID(ctx context.Context, accountID uuid.UUID) (uuid.UUID, error)
}

type repositoryService struct {
	db    *sql.DB
	redis *redis.Client
}

func NewRepositoryService(db *sql.DB, redis *redis.Client) RepositoryService {
	return &repositoryService{
		db:    db,
		redis: redis,
	}
}

func (rs *repositoryService) AccountExists(accountID uuid.UUID) error {
	query := "SELECT 1 FROM accounts WHERE id = $1 LIMIT 1"
	var exists int

	err := rs.db.QueryRow(query, accountID).Scan(&exists)
	if err == sql.ErrNoRows {
		return utils.NewNotFoundError(fmt.Sprintf("account %s not found", accountID), err)
	}
	if err != nil {
		return utils.NewInternalError(fmt.Errorf("db error: %w", err))
	}

	return nil
}

func (rs *repositoryService) CheckIdempotencyKey(ctx context.Context, ikey string) (string, error) {
	res, err := rs.redis.Get(ctx, ikey).Result()
	if err != nil {
		if err == redis.Nil {

			return "", nil
		}
		return "", utils.NewInternalError(fmt.Errorf("error while looking up key: %w", err))
	}

	return res, nil
}

func (rs *repositoryService) StoreIdempotencyKey(ctx context.Context, key string, data *models.IdempotencyCache, expiration time.Duration) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return utils.NewInternalError(fmt.Errorf("error marshaling cache data: %w", err))
	}

	err = rs.redis.Set(ctx, key, jsonData, expiration).Err()
	if err != nil {
		return utils.NewInternalError(fmt.Errorf("error storing in redis: %w", err))
	}

	return nil
}

func (rs *repositoryService) GetIdempotencyCache(ctx context.Context, key string) (*models.IdempotencyCache, error) {
	data, err := rs.redis.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil // key doesn't exist
		}
		return nil, utils.NewInternalError(fmt.Errorf("redis error: %w", err))
	}

	var cache models.IdempotencyCache
	if err := json.Unmarshal([]byte(data), &cache); err != nil {
		return nil, utils.NewInternalError(fmt.Errorf("error unmarshaling cache: %w", err))
	}

	return &cache, nil
}

func (rs *repositoryService) GetTransactionByIdempotencyKey(ctx context.Context, idempKey string) (*models.Transaction, error) {
	tx := &models.Transaction{}

	query := `SELECT id, idempotency_key, from_account_id, to_account_id, amount, currency, status, created_at, updated_at 
              FROM transactions 
              WHERE idempotency_key = $1 
              LIMIT 1`

	err := rs.db.QueryRowContext(ctx, query, idempKey).Scan(
		&tx.ID,
		&tx.IdempotencyKey,
		&tx.FromAccountID,
		&tx.ToAccountID,
		&tx.Amount,
		&tx.Currency,
		&tx.Status,
		&tx.CreatedAt,
		&tx.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // not found
		}
		return nil, utils.NewInternalError(fmt.Errorf("db error: %w", err))
	}

	return tx, nil
}

func (rs *repositoryService) CreateTransaction(ctx context.Context, tx *models.Transaction) (uuid.UUID, time.Time, error) {
	var transactionID uuid.UUID
	var timestamp time.Time

	q1 := `INSERT INTO transactions (idempotency_key, from_account_id, to_account_id, amount, currency, status, bank_reservation_id) 
           VALUES ($1, $2, $3, $4, $5, $6, $7) 
           RETURNING id, created_at`

	err := rs.db.QueryRowContext(ctx, q1,
		tx.IdempotencyKey,
		tx.FromAccountID,
		tx.ToAccountID, // this will correctly handle nil
		tx.Amount,
		tx.Currency,
		tx.Status,
		tx.ReservationID).Scan(&transactionID, &timestamp)

	if err != nil {
		return uuid.Nil, time.Time{}, utils.NewConstraintError(err)
	}

	tx.ID = transactionID
	tx.CreatedAt = timestamp
	return transactionID, timestamp, nil
}

func (rs *repositoryService) GetExternalBankAccountID(ctx context.Context, accountID uuid.UUID) (uuid.UUID, error) {
	var externalID uuid.UUID

	err := rs.db.QueryRowContext(ctx,
		"SELECT external_bank_account_id FROM accounts WHERE id = $1",
		accountID).Scan(&externalID)

	if err != nil {
		if err == sql.ErrNoRows {
			return uuid.Nil, utils.NewNotFoundError("account not found", err)
		}
		return uuid.Nil, utils.NewInternalError(err)
	}

	return externalID, nil
}
