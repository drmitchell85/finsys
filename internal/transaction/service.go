package transaction

import (
	"context"
	"fmt"
	"log"
	"log/slog"

	"github.com/drmitchell85/finsys/internal/bank"
	"github.com/drmitchell85/finsys/internal/messenger"
	"github.com/drmitchell85/finsys/internal/models"
	"github.com/drmitchell85/finsys/internal/store"
	"github.com/drmitchell85/finsys/internal/utils"
)

type TransactionService interface {
	CreateTransaction(ctx context.Context, req models.CreateTransactionRequest) (*models.CreateTransactionResponse, error)
	handleIdempotency(ctx context.Context, idempotencyKey string) (*models.CreateTransactionResponse, error)
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

	fmt.Println("CreateTransaction() called...")

	resp, err := ts.handleIdempotency(ctx, req.IdempotencyKey)
	if err != nil {
		return nil, err
	} else if resp != nil {
		return resp, nil
	}

	bankAccountID, err := ts.validateTransactionRequest(ctx, req)
	if err != nil {
		return nil, err
	}

	resID, err := ts.bs.ReserveFunds(ctx, bankAccountID, req.Amount)
	if err != nil {
		return nil, utils.WrapError(err, utils.ErrValidation, "error reserving funds")
	}

	resp, txID, err := ts.createAndCacheTransaction(ctx, req, resID)
	if err != nil {
		return nil, err
	}

	fmt.Println("moving on to enqueue...")

	// Enqueue for processing
	_, err = ts.qs.EnqueueTransaction(ctx, txID, req.IdempotencyKey, "default")
	if err != nil {
		return nil, utils.WrapError(err, utils.ErrInternal, "failed to enqueue transaction")
	}

	fmt.Println("checking queue...")

	mgs, err := ts.qs.ReceiveTransactions()
	if err != nil {
		log.Println("\nerr: \n", err)
	}

	fmt.Printf("mgs: %+v", mgs)

	return resp, nil
}
