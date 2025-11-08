package adapter_test

import (
	"context"
	"os"
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/m-mizutani/leveret/pkg/adapter"
)

func TestBigQuery(t *testing.T) {
	projectID := os.Getenv("TEST_BIGQUERY_PROJECT")
	if projectID == "" {
		t.Skip("TEST_BIGQUERY_PROJECT is not set")
	}

	datasetID := os.Getenv("TEST_BIGQUERY_DATASET")
	if datasetID == "" {
		t.Skip("TEST_BIGQUERY_DATASET is not set")
	}

	table := os.Getenv("TEST_BIGQUERY_TABLE")
	if table == "" {
		t.Skip("TEST_BIGQUERY_TABLE is not set")
	}

	query := os.Getenv("TEST_BIGQUERY_QUERY")
	if query == "" {
		t.Skip("TEST_BIGQUERY_QUERY is not set")
	}

	ctx := context.Background()
	client, err := adapter.NewBigQuery(ctx, projectID)
	gt.NoError(t, err)

	t.Run("DryRun", func(t *testing.T) {
		bytes, err := client.DryRun(ctx, query)
		gt.NoError(t, err)
		gt.True(t, bytes > 0)
		t.Logf("Bytes scanned: %d", bytes)
	})

	t.Run("Query", func(t *testing.T) {
		jobID, err := client.Query(ctx, query)
		gt.NoError(t, err)
		gt.NotEqual(t, "", jobID)
		t.Logf("Job ID: %s", jobID)

		// Get query result
		results, err := client.GetQueryResult(ctx, jobID)
		gt.NoError(t, err)
		gt.NotNil(t, results)
		t.Logf("Result count: %d", len(results))
	})

	t.Run("GetTableMetadata", func(t *testing.T) {
		metadata, err := client.GetTableMetadata(ctx, projectID, datasetID, table)
		gt.NoError(t, err)
		gt.NotNil(t, metadata)
		gt.NotNil(t, metadata.Schema)
		gt.True(t, len(metadata.Schema) > 0)

		t.Logf("Table: %s.%s.%s", projectID, datasetID, table)
		t.Logf("Fields: %d", len(metadata.Schema))
		t.Logf("Rows: %d", metadata.NumRows)
		t.Logf("Bytes: %d", metadata.NumBytes)

		if metadata.TimePartitioning != nil {
			t.Logf("Time partitioning: %s", metadata.TimePartitioning.Type)
			if metadata.TimePartitioning.Field != "" {
				t.Logf("  Field: %s", metadata.TimePartitioning.Field)
			}
		}

		if metadata.RangePartitioning != nil {
			t.Logf("Range partitioning field: %s", metadata.RangePartitioning.Field)
		}

		if len(metadata.Clustering.Fields) > 0 {
			t.Logf("Clustering fields: %v", metadata.Clustering.Fields)
		}
	})
}
