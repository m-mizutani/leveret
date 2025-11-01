package tool

import (
	"github.com/m-mizutani/leveret/pkg/adapter"
	"github.com/m-mizutani/leveret/pkg/repository"
)

// Client contains shared resources that tools can use
type Client struct {
	Repo    repository.Repository
	Gemini  adapter.Gemini
	Storage adapter.Storage
}
