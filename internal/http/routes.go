package http

import (
	"encoding/json"
	"net/http"

	"github.com/drmitchell85/finsys/internal/models"
	"github.com/go-chi/chi"
	"github.com/go-playground/validator"
)

func addRoutes(r *chi.Mux) {

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Ping!"))
	})

	r.Post("/transaction", createTransaction)

}

var validate = validator.New()

func createTransaction(w http.ResponseWriter, r *http.Request) {
	var reqObj models.CreateTransactionRequest
	err := json.NewDecoder(r.Body).Decode(&reqObj)
	if err != nil {
		respondFailure(w, 400, err)
		return
	}

	if err := validate.Struct(reqObj); err != nil {
		respondFailure(w, 400, err)
		return
	}

	// TODO now hand our reqObj off to the business logic...

	respondSuccess(w, 201, nil)
}
