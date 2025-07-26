package transaction

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/drmitchell85/finsys/internal/bank"
	"github.com/drmitchell85/finsys/internal/messenger"
	"github.com/drmitchell85/finsys/internal/models"
	"github.com/drmitchell85/finsys/internal/store"
	"github.com/drmitchell85/finsys/internal/utils"
	"github.com/shopspring/decimal"
)

type TransactionService interface {
	CreateTransaction(ctx context.Context, req models.CreateTransactionRequest) (*models.CreateTransactionResponse, error)
}

type transactionService struct {
	rs     store.RepositoryService
	qs     *messenger.QueueService
	bs     bank.BankService
	logger *slog.Logger
}

func NewTransactionService(rs store.RepositoryService, qs *messenger.QueueService, bs bank.BankService, logger *slog.Logger) TransactionService {
	return &transactionService{
		rs:     rs,
		qs:     qs,
		bs:     bs,
		logger: logger,
	}
}

func (ts *transactionService) CreateTransaction(ctx context.Context, req models.CreateTransactionRequest) (*models.CreateTransactionResponse, error) {

	data, err := ts.rs.CheckIdempotencyKey(ctx, req.IdempotencyKey)
	if err != nil {
		return nil, utils.WrapError(err, utils.ErrInternal, "error looking up cached key")
	}

	if data != "" {
		var cache models.IdempotencyCache

		err := json.Unmarshal([]byte(data), &cache)
		if err != nil {
			return nil, utils.NewInternalError(fmt.Errorf("error unmarshalling cached data: %w", err))
		}

		// Create a proper response from the cached data
		resp := &models.CreateTransactionResponse{
			TransactionID: cache.TransactionID,
			Status:        cache.Status,
			CreatedAt:     cache.CreatedAt,
		}

		// If the cache contains a full response object, unmarshal it
		if cache.Response != nil {
			var cachedResp models.CreateTransactionResponse
			if err := json.Unmarshal(cache.Response, &cachedResp); err != nil {
				// If we can't unmarshal the cached response, just use what we have
				return resp, nil
			}
			return &cachedResp, nil
		}

		return resp, nil
	}

	// Not in cache, check if already exists in DB
	existingTx, err := ts.rs.GetTransactionByIdempotencyKey(ctx, req.IdempotencyKey)
	if err != nil {
		return nil, utils.WrapError(err, utils.ErrInternal, "error checking existing transaction")
	}

	// If found in DB but not in cache, reconstruct and add to cache
	if existingTx != nil {
		resp := &models.CreateTransactionResponse{
			TransactionID: existingTx.ID,
			Status:        existingTx.Status,
			CreatedAt:     existingTx.CreatedAt,
		}

		// Re-cache the found transaction
		responseRaw, _ := json.Marshal(resp)
		err = ts.rs.StoreIdempotencyKey(ctx, req.IdempotencyKey, &models.IdempotencyCache{
			TransactionID: existingTx.ID,
			Status:        existingTx.Status,
			Response:      responseRaw,
			CreatedAt:     existingTx.CreatedAt,
		}, 24*time.Hour)
		if err != nil {
			ts.logger.Warn("failed to re-cache transaction", "error", err)
			// continue anyway
		}

		return resp, nil
	}

	err = ts.rs.AccountExists(req.FromAccountID)
	if err != nil {
		return nil, utils.WrapError(err, utils.ErrNotFound, "error while checking if account exists")
	}

	if req.ToAccountID != nil {
		err = ts.rs.AccountExists(*req.ToAccountID)
		if err != nil {
			return nil, utils.WrapError(err, utils.ErrNotFound, "error while checking if account exists")
		}
	}

	bankAccountID, err := ts.rs.GetExternalBankAccountID(ctx, req.FromAccountID)
	if err != nil {
		return nil, utils.WrapError(err, utils.ErrInternal, "failed to get external bank account")
	}

	hasFunds, err := ts.bs.HasSufficientFunds(ctx, bankAccountID, req.Amount)
	if err != nil {
		return nil, utils.WrapError(err, utils.ErrInternal, "failed to check account balance")
	} else if !hasFunds {
		return nil, utils.NewValidationError("insufficient funds", fmt.Errorf("funds error"))
	}

	err = validateCurrency(req.Currency, req.Amount)
	if err != nil {
		return nil, utils.WrapError(err, utils.ErrValidation, "error while validating currency")
	}

	resID, err := ts.bs.ReserveFunds(ctx, bankAccountID, req.Amount)
	if err != nil {
		return nil, utils.WrapError(err, utils.ErrValidation, "error reserving funds")
	}

	txID, txTime, err := ts.rs.CreateTransaction(ctx, &models.Transaction{
		IdempotencyKey: req.IdempotencyKey,
		FromAccountID:  req.FromAccountID,
		ToAccountID:    req.ToAccountID,
		Amount:         req.Amount,
		Currency:       req.Currency,
		Status:         models.TransactionPending,
		ReservationID:  resID,
	})

	if err != nil {
		// Check if it's a unique constraint violation
		var appErr *utils.AppError
		if errors.As(err, &appErr) && appErr.Code == utils.ErrUniqueConstraint {
			// Race condition - transaction was created between our check and insert
			// Try to fetch it again
			existingTx, fetchErr := ts.rs.GetTransactionByIdempotencyKey(ctx, req.IdempotencyKey)
			if fetchErr != nil {
				return nil, utils.WrapError(fetchErr, utils.ErrInternal, "failed to fetch constraint violation")
			}

			if existingTx != nil {
				// Use the existing transaction
				resp := &models.CreateTransactionResponse{
					TransactionID: existingTx.ID,
					Status:        existingTx.Status,
					CreatedAt:     existingTx.CreatedAt,
				}

				// Cache it
				responseRaw, _ := json.Marshal(resp)
				ts.rs.StoreIdempotencyKey(ctx, req.IdempotencyKey, &models.IdempotencyCache{
					TransactionID: existingTx.ID,
					Status:        existingTx.Status,
					Response:      responseRaw,
					CreatedAt:     existingTx.CreatedAt,
				}, 24*time.Hour)

				return resp, nil
			}

			return nil, utils.WrapError(err, utils.ErrInternal, "transaction exists but couldn't be retrieved")
		}

		return nil, utils.WrapError(err, utils.ErrInternal, "failed to create transaction entry")
	}

	resp := &models.CreateTransactionResponse{
		TransactionID: txID,
		Status:        models.TransactionPending,
		CreatedAt:     txTime,
	}

	// Cache the new transaction
	responseRaw, _ := json.Marshal(resp)
	err = ts.rs.StoreIdempotencyKey(ctx, req.IdempotencyKey, &models.IdempotencyCache{
		TransactionID: txID,
		Status:        models.TransactionPending,
		Response:      responseRaw,
		CreatedAt:     txTime,
	}, 24*time.Hour)
	if err != nil {
		ts.logger.Warn("failed to cache transaction", "error", err)
		// continue anyway
	}

	// Enqueue for processing
	_, err = ts.qs.EnqueueTransaction(ctx, txID, req.IdempotencyKey, "default")
	if err != nil {
		return nil, utils.WrapError(err, utils.ErrInternal, "failed to enqueue transaction")
	}

	return resp, nil
}

func validateCurrency(currency string, amount decimal.Decimal) error {
	if currency != "USD" {
		return utils.NewValidationError("only USD transactions supported", fmt.Errorf("error"))
	}

	// basic USD rules
	if amount.Exponent() < -2 {
		return utils.NewValidationError("USD amounts cannot have more than 2 decimal places", fmt.Errorf("error"))
	}

	minAmount := decimal.NewFromFloat(0.01) // 1 cent minimum
	if amount.LessThan(minAmount) {
		return utils.NewValidationError("minimum transaction amount is $0.01", fmt.Errorf("error"))
	}

	return nil
}
