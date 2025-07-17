include .env
export

.PHONY: run-trans
run-trans:
	go run cmd/server/main.go
