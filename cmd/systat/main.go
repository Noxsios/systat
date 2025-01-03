package main

import (
	"context"
	"os"

	"github.com/charmbracelet/log"
	"github.com/noxsios/systat/cmd"
)

func main() {
	logger := log.NewWithOptions(os.Stderr, log.Options{
		ReportTimestamp: false,
		ReportCaller:    false,
	})
	ctx := context.Background()
	ctx = log.WithContext(ctx, logger)

	if err := cmd.ExecuteContext(ctx); err != nil {
		logger.Print("")
		logger.Error(err)
		os.Exit(1)
	}
}
