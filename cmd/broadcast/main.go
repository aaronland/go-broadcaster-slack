package main

import (
	"context"
	_ "github.com/aaronland/go-broadcaster-slack"
	"github.com/aaronland/go-broadcaster/app/broadcast"
	"log"
)

func main() {

	ctx := context.Background()
	logger := log.Default()

	err := broadcast.Run(ctx, logger)

	if err != nil {
		logger.Fatalf("Failed to run broadcast application, %v", err)
	}
}
