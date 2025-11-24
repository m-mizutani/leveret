package workflow

import (
	"context"
	"os"
	"path/filepath"

	"github.com/m-mizutani/goerr/v2"
	"github.com/open-policy-agent/opa/v1/rego"
)

// loadPolicy loads a Rego policy file and prepares a query
func loadPolicy(ctx context.Context, policyPath, query string) (*rego.PreparedEvalQuery, error) {
	// Check if policy file exists
	if _, err := os.Stat(policyPath); err != nil {
		if os.IsNotExist(err) {
			return nil, nil // Policy file not found, return nil (caller should use default behavior)
		}
		return nil, goerr.Wrap(err, "failed to stat policy file", goerr.Value("path", policyPath))
	}

	// Read policy file
	data, err := os.ReadFile(policyPath)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to read policy file", goerr.Value("path", policyPath))
	}

	// Prepare Rego query
	r := rego.New(
		rego.Query(query),
		rego.Module(policyPath, string(data)),
	)

	prepared, err := r.PrepareForEval(ctx)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to prepare Rego query",
			goerr.Value("path", policyPath),
			goerr.Value("query", query))
	}

	return &prepared, nil
}

// loadPolicies loads all three phase policies
func loadPolicies(ctx context.Context, policyDir string) (ingest, enrich, triage *rego.PreparedEvalQuery, err error) {
	ingestPath := filepath.Join(policyDir, "ingest.rego")
	enrichPath := filepath.Join(policyDir, "enrich.rego")
	triagePath := filepath.Join(policyDir, "triage.rego")

	ingest, err = loadPolicy(ctx, ingestPath, "data.ingest")
	if err != nil {
		return nil, nil, nil, goerr.Wrap(err, "failed to load ingest policy")
	}

	enrich, err = loadPolicy(ctx, enrichPath, "data.enrich")
	if err != nil {
		return nil, nil, nil, goerr.Wrap(err, "failed to load enrich policy")
	}

	triage, err = loadPolicy(ctx, triagePath, "data.triage")
	if err != nil {
		return nil, nil, nil, goerr.Wrap(err, "failed to load triage policy")
	}

	return ingest, enrich, triage, nil
}
