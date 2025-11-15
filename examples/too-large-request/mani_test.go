package main_test

import (
	"os"
	"testing"

	main "github.com/m-mizutani/leveret/examples/too-large-request"
)

func TestRun(t *testing.T) {
	// Skip if GEMINI_PROJECT is not set
	if os.Getenv("GEMINI_PROJECT") == "" {
		t.Skip("GEMINI_PROJECT is not set")
	}

	if err := main.Run(t.Context()); err != nil {
		t.Fatal("expected no error", err)
	}
}
