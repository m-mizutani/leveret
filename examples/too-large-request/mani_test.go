package main_test

import (
	"testing"

	main "github.com/m-mizutani/leveret/examples/too-large-request"
)

func TestRun(t *testing.T) {
	if err := main.Run(t.Context()); err != nil {
		t.Fatal("expected no error", err)
	}
}
