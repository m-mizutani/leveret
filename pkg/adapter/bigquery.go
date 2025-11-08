package adapter

import (
	"context"

	"cloud.google.com/go/bigquery"
	"github.com/m-mizutani/goerr/v2"
	"google.golang.org/api/iterator"
)

// BigQuery is an interface for BigQuery operations
type BigQuery interface {
	// DryRun executes a query in dry-run mode and returns the number of bytes that will be scanned
	DryRun(ctx context.Context, query string) (int64, error)

	// Query executes a query and returns the job ID
	Query(ctx context.Context, query string) (string, error)

	// GetQueryResult retrieves the result of a query job
	GetQueryResult(ctx context.Context, jobID string) ([]map[string]any, error)

	// GetTableMetadata retrieves the metadata of a table including schema and partition information
	GetTableMetadata(ctx context.Context, project, datasetID, table string) (*bigquery.TableMetadata, error)
}

type bigqueryClient struct {
	client *bigquery.Client
}

// BigQueryOption is a functional option for BigQuery client
type BigQueryOption func(*bigqueryClient)

// NewBigQuery creates a new BigQuery client
func NewBigQuery(ctx context.Context, projectID string, opts ...BigQueryOption) (BigQuery, error) {
	client, err := bigquery.NewClient(ctx, projectID)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to create BigQuery client")
	}

	bq := &bigqueryClient{
		client: client,
	}

	for _, opt := range opts {
		opt(bq)
	}

	return bq, nil
}

// DryRun executes a query in dry-run mode and returns the number of bytes that will be scanned
func (bq *bigqueryClient) DryRun(ctx context.Context, query string) (int64, error) {
	q := bq.client.Query(query)
	q.DryRun = true

	job, err := q.Run(ctx)
	if err != nil {
		return 0, goerr.Wrap(err, "failed to run dry-run query")
	}

	status := job.LastStatus()
	if status == nil || status.Statistics == nil {
		return 0, goerr.New("no statistics available from dry-run")
	}

	return status.Statistics.TotalBytesProcessed, nil
}

// Query executes a query and returns the job ID
func (bq *bigqueryClient) Query(ctx context.Context, query string) (string, error) {
	q := bq.client.Query(query)

	job, err := q.Run(ctx)
	if err != nil {
		return "", goerr.Wrap(err, "failed to run query")
	}

	// Wait for the query to complete
	status, err := job.Wait(ctx)
	if err != nil {
		return "", goerr.Wrap(err, "failed to wait for query completion")
	}

	if status.Err() != nil {
		return "", goerr.Wrap(status.Err(), "query execution failed")
	}

	return job.ID(), nil
}

// GetQueryResult retrieves the result of a query job
func (bq *bigqueryClient) GetQueryResult(ctx context.Context, jobID string) ([]map[string]any, error) {
	// Try multiple approaches to get the job
	var job *bigquery.Job
	var err error

	// First, try JobFromID (works for jobs in the same location as client)
	job, err = bq.client.JobFromID(ctx, jobID)
	if err != nil {
		// If that fails, try common locations
		locations := []string{"us", "us-central1", "asia-northeast1", "europe-west1", "EU"}
		for _, loc := range locations {
			job, err = bq.client.JobFromIDLocation(ctx, jobID, loc)
			if err == nil {
				break
			}
		}
		if err != nil {
			return nil, goerr.Wrap(err, "failed to get job from ID (tried multiple locations)")
		}
	}

	// Read results from the job
	it, err := job.Read(ctx)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to read query result")
	}

	var results []map[string]any
	for {
		var row map[string]bigquery.Value
		err := it.Next(&row)
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, goerr.Wrap(err, "failed to iterate query result")
		}

		// Convert bigquery.Value to any
		rowMap := make(map[string]any)
		for k, v := range row {
			rowMap[k] = v
		}
		results = append(results, rowMap)
	}

	return results, nil
}

// GetTableMetadata retrieves the metadata of a table including schema and partition information
func (bq *bigqueryClient) GetTableMetadata(ctx context.Context, project, datasetID, table string) (*bigquery.TableMetadata, error) {
	// Create a client for the specified project if different from the current client's project
	client := bq.client
	if project != bq.client.Project() {
		var err error
		client, err = bigquery.NewClient(ctx, project)
		if err != nil {
			return nil, goerr.Wrap(err, "failed to create BigQuery client for project")
		}
		defer client.Close()
	}

	dataset := client.Dataset(datasetID)
	tbl := dataset.Table(table)

	metadata, err := tbl.Metadata(ctx)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to get table metadata")
	}

	return metadata, nil
}
