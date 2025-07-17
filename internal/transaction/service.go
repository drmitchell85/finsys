package transaction

import (
	"context"

	"github.com/drmitchell85/finsys/internal/models"
)

type TransactionService interface {
	CreateTransaction(ctx context.Context, req models.CreateTransactionRequest) (*models.CreateTransactionResponse, error)
}

type transactionService struct {
}

func NewTransactionService() TransactionService {
	return &transactionService{}
}

func (ts *transactionService) CreateTransaction(ctx context.Context, req models.CreateTransactionRequest) (*models.CreateTransactionResponse, error) {
	var res *models.CreateTransactionResponse

	// TODO check if accounts exist

	// TODO fromAccount for sufficient balance

	// TODO validate currencies

	// TODO check redis for existing transaction
	// if it does, return cached result

	// TODO enqueue transaction

	return res, nil
}
