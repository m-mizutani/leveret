package repository

import (
	"context"
	"sort"

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
	memoryCollection  = "memories"
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

func (r *Firestore) SearchAlerts(ctx context.Context, input *SearchAlertsInput) ([]*model.Alert, error) {
	client, err := r.getClient(ctx)
	if err != nil {
		return nil, err
	}

	// Set defaults
	if input.Limit <= 0 {
		input.Limit = 10
	}
	if input.Limit > 100 {
		input.Limit = 100
	}
	if input.Offset < 0 {
		input.Offset = 0
	}

	// Validate required fields
	if input.Field == "" {
		return nil, goerr.New("field is required")
	}
	if input.Operator == "" {
		return nil, goerr.New("operator is required")
	}

	// Automatically prefix field path with "Data."
	fieldPath := "Data." + input.Field

	// Build Firestore query
	query := client.Collection(alertCollection).
		Where(fieldPath, input.Operator, input.Value).
		Offset(input.Offset).
		Limit(input.Limit)

	// Execute query
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

func (r *Firestore) SearchSimilarAlerts(ctx context.Context, embedding []float32, threshold float64) ([]*model.Alert, error) {
	client, err := r.getClient(ctx)
	if err != nil {
		return nil, err
	}

	// Convert []float32 to firestore.Vector32
	vector32 := firestore.Vector32(embedding)

	// Build vector query with distance threshold
	query := client.Collection(alertCollection).
		FindNearest("Embedding", vector32, 1000, firestore.DistanceMeasureCosine, &firestore.FindNearestOptions{
			DistanceThreshold: &threshold,
		})

	// Execute query
	iter := query.Documents(ctx)
	defer iter.Stop()

	var alerts []*model.Alert
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, goerr.Wrap(err, "failed to iterate similar alerts")
		}

		var alert model.Alert
		if err := doc.DataTo(&alert); err != nil {
			return nil, goerr.Wrap(err, "failed to parse alert data", goerr.Value("id", doc.Ref.ID))
		}
		alerts = append(alerts, &alert)
	}

	return alerts, nil
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

func (r *Firestore) ListHistoryByAlert(ctx context.Context, alertID model.AlertID) ([]*model.History, error) {
	client, err := r.getClient(ctx)
	if err != nil {
		return nil, err
	}

	query := client.Collection(historyCollection).
		Where("AlertID", "==", alertID)

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

	// Sort in-memory by CreatedAt descending
	sort.Slice(histories, func(i, j int) bool {
		return histories[i].CreatedAt.After(histories[j].CreatedAt)
	})

	return histories, nil
}

func (r *Firestore) PutMemory(ctx context.Context, memory *model.Memory) error {
	client, err := r.getClient(ctx)
	if err != nil {
		return err
	}

	_, err = client.Collection(memoryCollection).Doc(string(memory.ID)).Set(ctx, memory)
	if err != nil {
		return goerr.Wrap(err, "failed to put memory", goerr.V("id", memory.ID))
	}

	return nil
}

func (r *Firestore) GetMemory(ctx context.Context, id model.MemoryID) (*model.Memory, error) {
	client, err := r.getClient(ctx)
	if err != nil {
		return nil, err
	}

	doc, err := client.Collection(memoryCollection).Doc(string(id)).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, goerr.Wrap(err, "memory not found", goerr.V("id", id))
		}
		return nil, goerr.Wrap(err, "failed to get memory", goerr.V("id", id))
	}

	var memory model.Memory
	if err := doc.DataTo(&memory); err != nil {
		return nil, goerr.Wrap(err, "failed to parse memory data", goerr.V("id", id))
	}

	return &memory, nil
}

func (r *Firestore) SearchMemories(ctx context.Context, embedding firestore.Vector32, threshold float64, limit int) ([]*model.Memory, error) {
	client, err := r.getClient(ctx)
	if err != nil {
		return nil, err
	}

	// Build vector query with distance threshold
	query := client.Collection(memoryCollection).
		FindNearest("Embedding", embedding, limit, firestore.DistanceMeasureCosine, &firestore.FindNearestOptions{
			DistanceThreshold: &threshold,
		})

	// Execute query
	iter := query.Documents(ctx)
	defer iter.Stop()

	var memories []*model.Memory
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, goerr.Wrap(err, "failed to iterate similar memories")
		}

		var memory model.Memory
		if err := doc.DataTo(&memory); err != nil {
			return nil, goerr.Wrap(err, "failed to parse memory data", goerr.V("id", doc.Ref.ID))
		}
		memories = append(memories, &memory)
	}

	return memories, nil
}

func (r *Firestore) UpdateMemoryScore(ctx context.Context, id model.MemoryID, delta float64) error {
	client, err := r.getClient(ctx)
	if err != nil {
		return err
	}

	docRef := client.Collection(memoryCollection).Doc(string(id))

	// Use transaction to atomically update score
	err = client.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		doc, err := tx.Get(docRef)
		if err != nil {
			if status.Code(err) == codes.NotFound {
				return goerr.Wrap(err, "memory not found", goerr.V("id", id))
			}
			return goerr.Wrap(err, "failed to get memory in transaction", goerr.V("id", id))
		}

		var memory model.Memory
		if err := doc.DataTo(&memory); err != nil {
			return goerr.Wrap(err, "failed to parse memory data", goerr.V("id", id))
		}

		// Update score and timestamp
		newScore := memory.Score + delta

		// Use Update instead of Set to update specific fields
		updates := []firestore.Update{
			{Path: "Score", Value: newScore},
			{Path: "UpdatedAt", Value: firestore.ServerTimestamp},
		}

		return tx.Update(docRef, updates)
	})

	if err != nil {
		return goerr.Wrap(err, "failed to update memory score", goerr.V("id", id), goerr.V("delta", delta))
	}

	return nil
}

func (r *Firestore) DeleteMemoriesBelowScore(ctx context.Context, threshold float64) error {
	client, err := r.getClient(ctx)
	if err != nil {
		return err
	}

	// Query for memories with score below threshold
	iter := client.Collection(memoryCollection).
		Where("Score", "<", threshold).
		Documents(ctx)
	defer iter.Stop()

	// Collect document references to delete
	var docsToDelete []*firestore.DocumentRef
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return goerr.Wrap(err, "failed to iterate memories to delete")
		}
		docsToDelete = append(docsToDelete, doc.Ref)
	}

	// Delete in batch
	if len(docsToDelete) > 0 {
		batch := client.Batch()
		for _, docRef := range docsToDelete {
			batch.Delete(docRef)
		}

		if _, err := batch.Commit(ctx); err != nil {
			return goerr.Wrap(err, "failed to delete memories", goerr.V("count", len(docsToDelete)), goerr.V("threshold", threshold))
		}
	}

	return nil
}
