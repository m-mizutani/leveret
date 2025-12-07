package repository_test

import (
	"context"
	"math/rand"
	"os"
	"testing"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/m-mizutani/gt"
	"github.com/m-mizutani/leveret/pkg/model"
	"github.com/m-mizutani/leveret/pkg/repository"
	"google.golang.org/genai"
)

func setupFirestore(t *testing.T) *repository.Firestore {
	projectID := os.Getenv("TEST_FIRESTORE_PROJECT_ID")
	databaseID := os.Getenv("TEST_FIRESTORE_DATABASE_ID")

	if projectID == "" || databaseID == "" {
		t.Skip("TEST_FIRESTORE_PROJECT_ID and TEST_FIRESTORE_DATABASE_ID must be set to run Firestore tests")
	}

	repo, err := repository.New(projectID, databaseID)
	gt.NoError(t, err)

	return repo
}

func TestFirestorePutAlert(t *testing.T) {
	repo := setupFirestore(t)
	ctx := context.Background()

	alert := &model.Alert{
		ID:          model.NewAlertID(),
		Title:       "Test Alert",
		Description: "This is a test alert",
		Data:        map[string]interface{}{"key": "value"},
		Attributes: []*model.Attribute{
			{
				Key:   "source_ip",
				Value: "192.168.1.1",
				Type:  model.AttributeTypeIPAddress,
			},
		},
		CreatedAt: time.Now(),
	}

	err := repo.PutAlert(ctx, alert)
	gt.NoError(t, err)
}

func TestFirestoreGetAlert(t *testing.T) {
	repo := setupFirestore(t)
	ctx := context.Background()

	// Put an alert first
	alert := &model.Alert{
		ID:          model.NewAlertID(),
		Title:       "Test Alert 2",
		Description: "This is another test alert",
		Data:        map[string]interface{}{"key": "value2"},
		Attributes: []*model.Attribute{
			{
				Key:   "destination_ip",
				Value: "10.0.0.1",
				Type:  model.AttributeTypeIPAddress,
			},
		},
		CreatedAt: time.Now(),
	}

	err := repo.PutAlert(ctx, alert)
	gt.NoError(t, err)

	// Get the alert
	retrieved, err := repo.GetAlert(ctx, alert.ID)
	gt.NoError(t, err)
	gt.V(t, retrieved).NotNil()
	gt.Equal(t, retrieved.ID, alert.ID)
	gt.Equal(t, retrieved.Title, alert.Title)
	gt.Equal(t, retrieved.Description, alert.Description)
}

func TestFirestoreGetAlertNotFound(t *testing.T) {
	repo := setupFirestore(t)
	ctx := context.Background()

	_, err := repo.GetAlert(ctx, model.AlertID("non-existent-alert"))
	gt.Error(t, err)
}

func TestFirestoreListAlerts(t *testing.T) {
	repo := setupFirestore(t)
	ctx := context.Background()

	// Put multiple alerts
	now := time.Now()
	alerts := []*model.Alert{
		{
			ID:          model.NewAlertID(),
			Title:       "List Alert 1",
			Description: "First list alert",
			CreatedAt:   now.Add(-2 * time.Hour),
		},
		{
			ID:          model.NewAlertID(),
			Title:       "List Alert 2",
			Description: "Second list alert",
			CreatedAt:   now.Add(-1 * time.Hour),
		},
		{
			ID:          model.NewAlertID(),
			Title:       "List Alert 3",
			Description: "Third list alert",
			CreatedAt:   now,
		},
	}

	for _, alert := range alerts {
		err := repo.PutAlert(ctx, alert)
		gt.NoError(t, err)
	}

	// List alerts with limit - just verify we got results and they're ordered
	retrieved, err := repo.ListAlerts(ctx, 0, 10)
	gt.NoError(t, err)
	gt.A(t, retrieved).Longer(2)

	// Verify ordering: CreatedAt should be descending
	if len(retrieved) >= 2 {
		for i := 0; i < len(retrieved)-1; i++ {
			if !retrieved[i].CreatedAt.After(retrieved[i+1].CreatedAt) && !retrieved[i].CreatedAt.Equal(retrieved[i+1].CreatedAt) {
				t.Errorf("alerts not properly ordered: [%d].CreatedAt (%v) should be >= [%d].CreatedAt (%v)",
					i, retrieved[i].CreatedAt, i+1, retrieved[i+1].CreatedAt)
			}
		}
	}
}

func TestFirestoreListAlertsEmpty(t *testing.T) {
	repo := setupFirestore(t)
	ctx := context.Background()

	// List with large offset should return empty
	retrieved, err := repo.ListAlerts(ctx, 10000, 10)
	gt.NoError(t, err)
	gt.A(t, retrieved).Length(0)
}

func TestFirestorePutHistory(t *testing.T) {
	repo := setupFirestore(t)
	ctx := context.Background()

	alertID := model.NewAlertID()
	history := &model.History{
		ID:        model.NewHistoryID(),
		Title:     "Test Conversation",
		AlertID:   alertID,
		Contents:  []*genai.Content{},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err := repo.PutHistory(ctx, history)
	gt.NoError(t, err)
}

func TestFirestoreGetHistory(t *testing.T) {
	repo := setupFirestore(t)
	ctx := context.Background()

	// Put a history first
	alertID := model.NewAlertID()
	history := &model.History{
		ID:        model.NewHistoryID(),
		Title:     "Test Conversation 2",
		AlertID:   alertID,
		Contents:  []*genai.Content{},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err := repo.PutHistory(ctx, history)
	gt.NoError(t, err)

	// Get the history
	retrieved, err := repo.GetHistory(ctx, history.ID)
	gt.NoError(t, err)
	gt.V(t, retrieved).NotNil()
	gt.Equal(t, retrieved.ID, history.ID)
	gt.Equal(t, retrieved.Title, history.Title)
	gt.Equal(t, retrieved.AlertID, history.AlertID)
}

func TestFirestoreGetHistoryNotFound(t *testing.T) {
	repo := setupFirestore(t)
	ctx := context.Background()

	_, err := repo.GetHistory(ctx, model.HistoryID("non-existent-history"))
	gt.Error(t, err)
}

func TestFirestoreListHistory(t *testing.T) {
	repo := setupFirestore(t)
	ctx := context.Background()

	// Put multiple histories
	now := time.Now()
	alertID := model.NewAlertID()
	histories := []*model.History{
		{
			ID:        model.NewHistoryID(),
			Title:     "Conversation 1",
			AlertID:   alertID,
			Contents:  []*genai.Content{},
			CreatedAt: now.Add(-2 * time.Hour),
			UpdatedAt: now.Add(-2 * time.Hour),
		},
		{
			ID:        model.NewHistoryID(),
			Title:     "Conversation 2",
			AlertID:   alertID,
			Contents:  []*genai.Content{},
			CreatedAt: now.Add(-1 * time.Hour),
			UpdatedAt: now.Add(-1 * time.Hour),
		},
		{
			ID:        model.NewHistoryID(),
			Title:     "Conversation 3",
			AlertID:   alertID,
			Contents:  []*genai.Content{},
			CreatedAt: now,
			UpdatedAt: now,
		},
	}

	for _, history := range histories {
		err := repo.PutHistory(ctx, history)
		gt.NoError(t, err)
	}

	// List histories with limit - just verify we got results and they're ordered
	retrieved, err := repo.ListHistory(ctx, 0, 10)
	gt.NoError(t, err)
	gt.A(t, retrieved).Longer(2)

	// Verify ordering: CreatedAt should be descending
	if len(retrieved) >= 2 {
		for i := 0; i < len(retrieved)-1; i++ {
			if !retrieved[i].CreatedAt.After(retrieved[i+1].CreatedAt) && !retrieved[i].CreatedAt.Equal(retrieved[i+1].CreatedAt) {
				t.Errorf("histories not properly ordered: [%d].CreatedAt (%v) should be >= [%d].CreatedAt (%v)",
					i, retrieved[i].CreatedAt, i+1, retrieved[i+1].CreatedAt)
			}
		}
	}
}

func TestFirestoreListHistoryEmpty(t *testing.T) {
	repo := setupFirestore(t)
	ctx := context.Background()

	// List with large offset should return empty
	retrieved, err := repo.ListHistory(ctx, 10000, 10)
	gt.NoError(t, err)
	gt.A(t, retrieved).Length(0)
}

func TestFirestoreSearchAlerts(t *testing.T) {
	repo := setupFirestore(t)
	ctx := context.Background()

	// Put test alerts with different data
	now := time.Now()
	alerts := []*model.Alert{
		{
			ID:          model.NewAlertID(),
			Title:       "Critical Alert",
			Description: "High severity security alert",
			Data: map[string]interface{}{
				"severity": "critical",
				"source":   "guardduty",
				"type":     "UnauthorizedAccess:EC2/SSHBruteForce",
				"score":    9,
			},
			CreatedAt: now.Add(-3 * time.Hour),
		},
		{
			ID:          model.NewAlertID(),
			Title:       "High Alert",
			Description: "Medium severity security alert",
			Data: map[string]interface{}{
				"severity": "high",
				"source":   "securityhub",
				"type":     "UnauthorizedAccess:EC2/RDPBruteForce",
				"score":    7,
			},
			CreatedAt: now.Add(-2 * time.Hour),
		},
		{
			ID:          model.NewAlertID(),
			Title:       "Medium Alert",
			Description: "Low severity security alert",
			Data: map[string]interface{}{
				"severity": "medium",
				"source":   "guardduty",
				"type":     "Recon:EC2/PortProbeUnprotectedPort",
				"score":    5,
			},
			CreatedAt: now.Add(-1 * time.Hour),
		},
	}

	for _, alert := range alerts {
		err := repo.PutAlert(ctx, alert)
		gt.NoError(t, err)
	}

	// Wait a bit for Firestore to index
	time.Sleep(2 * time.Second)

	t.Run("search by severity string equality", func(t *testing.T) {
		results, err := repo.SearchAlerts(ctx, &repository.SearchAlertsInput{
			Field:    "severity",
			Operator: "==",
			Value:    "critical",
			Limit:    10,
			Offset:   0,
		})
		gt.NoError(t, err)
		gt.A(t, results).Longer(0) // At least one result
		// Verify all results have severity=critical
		for _, r := range results {
			if data, ok := r.Data.(map[string]interface{}); ok {
				gt.Equal(t, data["severity"], "critical")
			}
		}
	})

	t.Run("search by source", func(t *testing.T) {
		results, err := repo.SearchAlerts(ctx, &repository.SearchAlertsInput{
			Field:    "source",
			Operator: "==",
			Value:    "guardduty",
			Limit:    10,
			Offset:   0,
		})
		gt.NoError(t, err)
		gt.A(t, results).Longer(0) // At least one result
		// Verify all results have source=guardduty
		for _, r := range results {
			if data, ok := r.Data.(map[string]interface{}); ok {
				gt.Equal(t, data["source"], "guardduty")
			}
		}
	})

	t.Run("search by numeric score greater than", func(t *testing.T) {
		results, err := repo.SearchAlerts(ctx, &repository.SearchAlertsInput{
			Field:    "score",
			Operator: ">",
			Value:    6,
			Limit:    10,
			Offset:   0,
		})
		gt.NoError(t, err)
		gt.A(t, results).Longer(0) // At least one result
		// Verify all results have score > 6
		for _, r := range results {
			if data, ok := r.Data.(map[string]interface{}); ok {
				score := data["score"]
				// Firestore may return as float64 or int64
				switch v := score.(type) {
				case float64:
					if v <= 6 {
						t.Errorf("expected score > 6, got %v", v)
					}
				case int64:
					if v <= 6 {
						t.Errorf("expected score > 6, got %v", v)
					}
				}
			}
		}
	})

	t.Run("search with limit", func(t *testing.T) {
		results, err := repo.SearchAlerts(ctx, &repository.SearchAlertsInput{
			Field:    "source",
			Operator: "==",
			Value:    "guardduty",
			Limit:    1,
			Offset:   0,
		})
		gt.NoError(t, err)
		if len(results) > 1 {
			t.Errorf("expected at most 1 result, got %d", len(results))
		}
	})

	t.Run("search with no results", func(t *testing.T) {
		results, err := repo.SearchAlerts(ctx, &repository.SearchAlertsInput{
			Field:    "severity",
			Operator: "==",
			Value:    "nonexistent",
			Limit:    10,
			Offset:   0,
		})
		gt.NoError(t, err)
		gt.A(t, results).Length(0)
	})
}

func TestFirestoreSearchAlertsValidation(t *testing.T) {
	repo := setupFirestore(t)
	ctx := context.Background()

	t.Run("missing field", func(t *testing.T) {
		_, err := repo.SearchAlerts(ctx, &repository.SearchAlertsInput{
			Field:    "",
			Operator: "==",
			Value:    "test",
		})
		gt.Error(t, err)
	})

	t.Run("missing operator", func(t *testing.T) {
		_, err := repo.SearchAlerts(ctx, &repository.SearchAlertsInput{
			Field:    "severity",
			Operator: "",
			Value:    "test",
		})
		gt.Error(t, err)
	})

	t.Run("default limit", func(t *testing.T) {
		// Should not error with limit=0 (uses default)
		_, err := repo.SearchAlerts(ctx, &repository.SearchAlertsInput{
			Field:    "severity",
			Operator: "==",
			Value:    "test",
			Limit:    0, // Should default to 10
		})
		gt.NoError(t, err)
	})

	t.Run("limit exceeds maximum", func(t *testing.T) {
		// Should cap at 100
		_, err := repo.SearchAlerts(ctx, &repository.SearchAlertsInput{
			Field:    "severity",
			Operator: "==",
			Value:    "test",
			Limit:    200, // Should be capped to 100
		})
		gt.NoError(t, err)
	})
}

func TestFirestoreAlertEmbedding(t *testing.T) {
	repo := setupFirestore(t)
	ctx := context.Background()

	// Create test embedding vector
	testEmbedding := make(firestore.Vector32, 768)
	for i := range testEmbedding {
		testEmbedding[i] = float32(i) / 768.0
	}

	// Put an alert with embedding
	alert := &model.Alert{
		ID:          model.NewAlertID(),
		Title:       "Test Alert with Embedding",
		Description: "This alert has an embedding vector",
		Data:        map[string]interface{}{"key": "value"},
		Embedding:   testEmbedding,
		CreatedAt:   time.Now(),
	}

	err := repo.PutAlert(ctx, alert)
	gt.NoError(t, err)

	// Get the alert back
	retrieved, err := repo.GetAlert(ctx, alert.ID)
	gt.NoError(t, err)
	gt.V(t, retrieved).NotNil()
	gt.Equal(t, retrieved.ID, alert.ID)
	gt.A(t, retrieved.Embedding).Length(768)

	// Verify embedding values
	for i := 0; i < 768; i++ {
		if retrieved.Embedding[i] != testEmbedding[i] {
			t.Errorf("embedding mismatch at index %d: expected %v, got %v",
				i, testEmbedding[i], retrieved.Embedding[i])
			break
		}
	}
}

func TestFirestoreSearchSimilarAlerts(t *testing.T) {
	repo := setupFirestore(t)
	ctx := context.Background()

	// Create alerts with different embedding vectors
	now := time.Now()
	rng := rand.New(rand.NewSource(now.UnixNano()))

	// Alert 1: embedding with random values around 0.5
	embedding1 := make(firestore.Vector32, 768)
	for i := range embedding1 {
		embedding1[i] = 0.5 + float32(rng.Float64()*0.02-0.01) // 0.49-0.51
	}
	alert1 := &model.Alert{
		ID:          model.NewAlertID(),
		Title:       "Alert 1",
		Description: "Similar to query",
		Data:        map[string]interface{}{"type": "type_a"},
		Embedding:   embedding1,
		CreatedAt:   now.Add(-3 * time.Hour),
	}

	// Alert 2: embedding with random values around 0.5 (very similar to alert1)
	embedding2 := make(firestore.Vector32, 768)
	for i := range embedding2 {
		embedding2[i] = 0.5 + float32(rng.Float64()*0.02-0.01) // 0.49-0.51
	}
	alert2 := &model.Alert{
		ID:          model.NewAlertID(),
		Title:       "Alert 2",
		Description: "Also similar to query",
		Data:        map[string]interface{}{"type": "type_a"},
		Embedding:   embedding2,
		CreatedAt:   now.Add(-2 * time.Hour),
	}

	// Alert 3: embedding with random values very different from alert1 and alert2
	embedding3 := make(firestore.Vector32, 768)
	for i := range embedding3 {
		embedding3[i] = 0.9 + float32(rng.Float64()*0.02-0.01) // 0.89-0.91
	}
	alert3 := &model.Alert{
		ID:          model.NewAlertID(),
		Title:       "Alert 3",
		Description: "Different from query",
		Data:        map[string]interface{}{"type": "type_b"},
		Embedding:   embedding3,
		CreatedAt:   now.Add(-1 * time.Hour),
	}

	// Put all alerts
	for _, alert := range []*model.Alert{alert1, alert2, alert3} {
		err := repo.PutAlert(ctx, alert)
		gt.NoError(t, err)
	}

	// Search with query embedding similar to embedding1 and embedding2
	queryEmbedding := make([]float64, 768)
	for i := range queryEmbedding {
		queryEmbedding[i] = 0.5 + (rng.Float64()*0.02 - 0.01) // 0.49-0.51
	}

	results, err := repo.SearchSimilarAlerts(ctx, queryEmbedding, 3)
	gt.NoError(t, err)
	gt.A(t, results).Longer(0) // At least one result

	// The first results should be alert1 and alert2 (most similar)
	// Note: Exact ordering depends on Firestore's vector search implementation
	foundAlert1 := false
	foundAlert2 := false
	for _, r := range results {
		if r.ID == alert1.ID {
			foundAlert1 = true
		}
		if r.ID == alert2.ID {
			foundAlert2 = true
		}
	}

	// At least one of the similar alerts should be in the results
	if !foundAlert1 && !foundAlert2 {
		t.Error("expected to find at least one similar alert in results")
	}
}

func TestFirestoreSearchSimilarAlertsLimit(t *testing.T) {
	repo := setupFirestore(t)
	ctx := context.Background()

	// Create multiple alerts with embeddings
	now := time.Now()
	rng := rand.New(rand.NewSource(now.UnixNano()))
	for i := 0; i < 5; i++ {
		embedding := make(firestore.Vector32, 768)
		for j := range embedding {
			// Add random variation to avoid identical vectors
			base := float32(i) / 10.0
			embedding[j] = base + float32(rng.Float64()*0.01-0.005)
		}
		alert := &model.Alert{
			ID:          model.NewAlertID(),
			Title:       "Alert " + string(rune('A'+i)),
			Description: "Test alert",
			Data:        map[string]interface{}{"index": i},
			Embedding:   embedding,
			CreatedAt:   now.Add(time.Duration(-i) * time.Hour),
		}
		err := repo.PutAlert(ctx, alert)
		gt.NoError(t, err)
	}

	// Wait for indexing
	time.Sleep(3 * time.Second)

	// Search with limit=2
	queryEmbedding := make([]float64, 768)
	for i := range queryEmbedding {
		queryEmbedding[i] = 0.1 + (rng.Float64()*0.02 - 0.01) // 0.09-0.11, similar to first alert
	}

	results, err := repo.SearchSimilarAlerts(ctx, queryEmbedding, 2)
	gt.NoError(t, err)
	if len(results) > 2 {
		t.Errorf("expected at most 2 results, got %d", len(results))
	}
}
