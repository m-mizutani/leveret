package workflow_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/m-mizutani/leveret/pkg/workflow"
)

func TestIngestPhase(t *testing.T) {
	ctx := context.Background()

	// Create temporary policy directory
	tmpDir := t.TempDir()

	// Write ingest policy
	ingestPolicy := `package ingest

alert contains {
	"title": "Test Alert",
	"description": "Test Description",
	"attributes": [],
} if {
	input.test == true
}
`
	gt.NoError(t, os.WriteFile(filepath.Join(tmpDir, "ingest.rego"), []byte(ingestPolicy), 0644))

	// Create engine
	engine, err := workflow.New(ctx, tmpDir, nil, nil)
	gt.NoError(t, err)

	// Test data that matches policy
	data := map[string]any{
		"test": true,
	}

	results, err := engine.Execute(ctx, data)
	gt.NoError(t, err)
	gt.Equal(t, len(results), 1)
	gt.Equal(t, results[0].Alert.Title, "Test Alert")

	// Test data that doesn't match policy
	data2 := map[string]any{
		"test": false,
	}

	results2, err := engine.Execute(ctx, data2)
	gt.NoError(t, err)
	gt.Equal(t, len(results2), 0)
}

func TestTriagePhase(t *testing.T) {
	ctx := context.Background()

	// Create temporary policy directory
	tmpDir := t.TempDir()

	// Write minimal ingest policy
	ingestPolicy := `package ingest

alert contains {
	"title": input.title,
	"description": "",
	"attributes": [],
} if {
	input.title != ""
}
`
	gt.NoError(t, os.WriteFile(filepath.Join(tmpDir, "ingest.rego"), []byte(ingestPolicy), 0644))

	// Write triage policy
	triagePolicy := `package triage

default action = "accept"
default severity = "medium"
default note = ""

action = "discard" if {
	contains(input.alert.title, "maintenance")
}

severity = "critical" if {
	contains(input.alert.title, "critical")
}
`
	gt.NoError(t, os.WriteFile(filepath.Join(tmpDir, "triage.rego"), []byte(triagePolicy), 0644))

	// Create engine
	engine, err := workflow.New(ctx, tmpDir, nil, nil)
	gt.NoError(t, err)

	// Test critical alert
	data := map[string]any{
		"title": "critical issue detected",
	}

	results, err := engine.Execute(ctx, data)
	gt.NoError(t, err)
	gt.Equal(t, len(results), 1)
	gt.Equal(t, results[0].Triage.Severity, "critical")
	gt.Equal(t, results[0].Triage.Action, "accept")

	// Test maintenance alert (should be discarded)
	data2 := map[string]any{
		"title": "scheduled maintenance",
	}

	results2, err := engine.Execute(ctx, data2)
	gt.NoError(t, err)
	gt.Equal(t, len(results2), 1)
	gt.Equal(t, results2[0].Triage.Action, "discard")
}

func TestNoPolicyFiles(t *testing.T) {
	ctx := context.Background()

	// Create empty temporary directory
	tmpDir := t.TempDir()

	// Create engine without policy files
	engine, err := workflow.New(ctx, tmpDir, nil, nil)
	gt.NoError(t, err)

	// Should use default behavior
	data := map[string]any{
		"test": true,
	}

	results, err := engine.Execute(ctx, data)
	gt.NoError(t, err)
	// Without ingest policy, no alerts should be generated
	gt.Equal(t, len(results), 0)
}
