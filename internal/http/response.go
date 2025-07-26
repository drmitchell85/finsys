// internal/http/response.go
package http

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/drmitchell85/finsys/internal/utils"
)

type Response struct {
	Success bool           `json:"success"`
	Data    interface{}    `json:"data,omitempty"`
	Error   *ErrorResponse `json:"error,omitempty"`
}

type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func respondSuccess(w http.ResponseWriter, code int, payload interface{}) {
	res := Response{
		Success: true,
		Data:    payload,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(res)
}

func respondError(w http.ResponseWriter, err error) {
	// default to internal server error
	code := http.StatusInternalServerError
	errorResponse := ErrorResponse{
		Code:    string(utils.ErrInternal),
		Message: "An unexpected error occurred",
	}

	// check if it's our custom error type
	if appErr, ok := utils.GetAppError(err); ok {
		// map error codes to http status codes
		switch appErr.Code {
		case utils.ErrValidation:
			code = http.StatusBadRequest
		case utils.ErrNotFound:
			code = http.StatusNotFound
		case utils.ErrUnauthorized:
			code = http.StatusUnauthorized
		case utils.ErrForbidden:
			code = http.StatusForbidden
		case utils.ErrInsufficientFunds, utils.ErrAccountNotFound, utils.ErrDuplicateRequest:
			code = http.StatusBadRequest
		default:
			// log unknown app errors at error level
			log.Printf("ERROR: %v", err)
		}

		errorResponse.Code = string(appErr.Code)
		errorResponse.Message = appErr.Message
	} else {
		// not our error type, log it
		log.Printf("UNEXPECTED ERROR: %v", err)
	}

	res := Response{
		Success: false,
		Error:   &errorResponse,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(res)
}
