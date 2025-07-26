package http

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/drmitchell85/finsys/internal/models"
	"github.com/drmitchell85/finsys/internal/transaction"
	"github.com/go-chi/chi"
	"github.com/go-playground/validator"
)

func addRoutes(r *chi.Mux, ts transaction.TransactionService, ctx context.Context) {

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Ping!"))
	})

	r.Post("/transaction", createTransactionHandler(ts, ctx))

}

var validate = validator.New()

func createTransactionHandler(ts transaction.TransactionService, ctx context.Context) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var reqObj models.CreateTransactionRequest
		err := json.NewDecoder(r.Body).Decode(&reqObj)
		if err != nil {
			respondError(w, err)
			return
		}

		if err := validate.Struct(reqObj); err != nil {
			respondError(w, err)
			return
		}

		_, err = ts.CreateTransaction(ctx, reqObj)
		if err != nil {
			respondError(w, err)
			return
		}

		respondSuccess(w, 201, nil)
	}
}
