package workflow

import (
	"context"
	"os"
	"path/filepath"

	"github.com/m-mizutani/goerr/v2"
	"github.com/open-policy-agent/opa/v1/rego"
)

// loadPolicies loads all Rego files from policyDir and prepares queries for each phase
func loadPolicies(ctx context.Context, policyDir string) (ingest, enrich, triage *rego.PreparedEvalQuery, err error) {
	// Read all .rego files from the directory
	files, err := filepath.Glob(filepath.Join(policyDir, "*.rego"))
	if err != nil {
		return nil, nil, nil, goerr.Wrap(err, "failed to glob policy files")
	}

	if len(files) == 0 {
		// No policy files found, return nil for all phases
		return nil, nil, nil, nil
	}

	// Load all policy files as modules
	modules := make([]func(*rego.Rego), 0, len(files))
	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			return nil, nil, nil, goerr.Wrap(err, "failed to read policy file", goerr.Value("path", file))
		}
		modules = append(modules, rego.Module(file, string(data)))
	}

	// Prepare query for ingest phase
	ingest, err = prepareQuery(ctx, modules, "data.ingest")
	if err != nil {
		return nil, nil, nil, goerr.Wrap(err, "failed to prepare ingest query")
	}

	// Prepare query for enrich phase
	enrich, err = prepareQuery(ctx, modules, "data.enrich")
	if err != nil {
		return nil, nil, nil, goerr.Wrap(err, "failed to prepare enrich query")
	}

	// Prepare query for triage phase
	triage, err = prepareQuery(ctx, modules, "data.triage")
	if err != nil {
		return nil, nil, nil, goerr.Wrap(err, "failed to prepare triage query")
	}

	return ingest, enrich, triage, nil
}

// prepareQuery prepares a Rego query with all loaded modules
func prepareQuery(ctx context.Context, modules []func(*rego.Rego), query string) (*rego.PreparedEvalQuery, error) {
	// Build Rego options
	options := make([]func(*rego.Rego), 0, len(modules)+1)
	options = append(options, rego.Query(query))
	options = append(options, modules...)

	r := rego.New(options...)

	prepared, err := r.PrepareForEval(ctx)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to prepare query", goerr.Value("query", query))
	}

	return &prepared, nil
}
