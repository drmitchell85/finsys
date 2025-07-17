package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/drmitchell85/finsys/internal/http"
)

func main() {
	transactionServer, err := http.NewServer()
	if err != nil {
		log.Fatalf("Error starting transaction server: %s", err)
	}

	errc := make(chan error)
	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, syscall.SIGTERM, syscall.SIGINT)

	go func() {
		err = transactionServer.Start()
		if err != nil {
			log.Fatalf("transaction service: failed to start: %v", err)
			errc <- err
		}
	}()

	select {
	case <-sigc:
		log.Println("received signal to shut down transaction service...")
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		transactionServer.Shutdown(ctx)
		cancel()

	case <-errc:
		log.Println("error starting up transaction service, shutting down...")
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		transactionServer.Shutdown(ctx)
		cancel()
	}

	os.Exit(0)
}
