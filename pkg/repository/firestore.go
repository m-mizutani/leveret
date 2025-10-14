package repository

import (
	"context"

	"cloud.google.com/go/firestore"
	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/leveret/pkg/model"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	alertCollection   = "alerts"
	historyCollection = "histories"
)

// Firestore implements Repository interface using Firestore
type Firestore struct {
	projectID    string
	databaseName string
	client       *firestore.Client
}

var _ Repository = &Firestore{}

// NewFirestore creates a new Firestore repository
func New(projectID, databaseName string) (*Firestore, error) {
	return &Firestore{
		projectID:    projectID,
		databaseName: databaseName,
	}, nil
}

func (r *Firestore) getClient(ctx context.Context) (*firestore.Client, error) {
	if r.client != nil {
		return r.client, nil
	}

	client, err := firestore.NewClientWithDatabase(ctx, r.projectID, r.databaseName)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to create Firestore client")
	}
	r.client = client
	return client, nil
}

func (r *Firestore) PutAlert(ctx context.Context, alert *model.Alert) error {
	client, err := r.getClient(ctx)
	if err != nil {
		return err
	}

	_, err = client.Collection(alertCollection).Doc(string(alert.ID)).Set(ctx, alert)
	if err != nil {
		return goerr.Wrap(err, "failed to put alert", goerr.Value("id", alert.ID))
	}

	return nil
}

func (r *Firestore) GetAlert(ctx context.Context, id model.AlertID) (*model.Alert, error) {
	client, err := r.getClient(ctx)
	if err != nil {
		return nil, err
	}

	doc, err := client.Collection(alertCollection).Doc(string(id)).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, goerr.Wrap(err, "alert not found", goerr.Value("id", id))
		}
		return nil, goerr.Wrap(err, "failed to get alert", goerr.Value("id", id))
	}

	var alert model.Alert
	if err := doc.DataTo(&alert); err != nil {
		return nil, goerr.Wrap(err, "failed to parse alert data", goerr.Value("id", id))
	}

	return &alert, nil
}

func (r *Firestore) ListAlerts(ctx context.Context, offset, limit int) ([]*model.Alert, error) {
	client, err := r.getClient(ctx)
	if err != nil {
		return nil, err
	}

	query := client.Collection(alertCollection).
		OrderBy("CreatedAt", firestore.Desc).
		Offset(offset).
		Limit(limit)

	iter := query.Documents(ctx)
	defer iter.Stop()

	var alerts []*model.Alert
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, goerr.Wrap(err, "failed to iterate alerts")
		}

		var alert model.Alert
		if err := doc.DataTo(&alert); err != nil {
			return nil, goerr.Wrap(err, "failed to parse alert data", goerr.Value("id", doc.Ref.ID))
		}
		alerts = append(alerts, &alert)
	}

	return alerts, nil
}

func (r *Firestore) SearchSimilarAlerts(ctx context.Context, embedding []float64, limit int) ([]*model.Alert, error) {
	// TODO: Implement Firestore vector search
	return nil, nil
}

func (r *Firestore) PutHistory(ctx context.Context, history *model.History) error {
	client, err := r.getClient(ctx)
	if err != nil {
		return err
	}

	_, err = client.Collection(historyCollection).Doc(string(history.ID)).Set(ctx, history)
	if err != nil {
		return goerr.Wrap(err, "failed to put history", goerr.Value("id", history.ID))
	}

	return nil
}

func (r *Firestore) GetHistory(ctx context.Context, id model.HistoryID) (*model.History, error) {
	client, err := r.getClient(ctx)
	if err != nil {
		return nil, err
	}

	doc, err := client.Collection(historyCollection).Doc(string(id)).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, goerr.Wrap(err, "history not found", goerr.Value("id", id))
		}
		return nil, goerr.Wrap(err, "failed to get history", goerr.Value("id", id))
	}

	var history model.History
	if err := doc.DataTo(&history); err != nil {
		return nil, goerr.Wrap(err, "failed to parse history data", goerr.Value("id", id))
	}

	return &history, nil
}

func (r *Firestore) ListHistory(ctx context.Context, offset, limit int) ([]*model.History, error) {
	client, err := r.getClient(ctx)
	if err != nil {
		return nil, err
	}

	query := client.Collection(historyCollection).
		OrderBy("CreatedAt", firestore.Desc).
		Offset(offset).
		Limit(limit)

	iter := query.Documents(ctx)
	defer iter.Stop()

	var histories []*model.History
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, goerr.Wrap(err, "failed to iterate histories")
		}

		var history model.History
		if err := doc.DataTo(&history); err != nil {
			return nil, goerr.Wrap(err, "failed to parse history data", goerr.Value("id", doc.Ref.ID))
		}
		histories = append(histories, &history)
	}

	return histories, nil
}
