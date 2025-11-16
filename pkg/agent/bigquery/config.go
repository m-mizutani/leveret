package bigquery

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/m-mizutani/goerr/v2"
	"gopkg.in/yaml.v3"
)

// runBook represents a SQL runBook with metadata
type runBook struct {
	ID          string
	Title       string
	Description string
	FilePath    string
	Query       string
}

// loadRunBooks scans a directory and loads all SQL files as runBooks
func loadRunBooks(dir string) (map[string]*runBook, error) {
	if dir == "" {
		return nil, nil
	}

	// Check if directory exists
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return nil, goerr.Wrap(err, "runbook directory does not exist", goerr.V("dir", dir))
	}

	runBooks := make(map[string]*runBook)

	// Scan directory for .sql files
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to read runbook directory", goerr.V("dir", dir))
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		if !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}

		filePath := filepath.Join(dir, entry.Name())

		// Read file content
		content, err := os.ReadFile(filePath)
		if err != nil {
			return nil, goerr.Wrap(err, "failed to read runbook file", goerr.V("file", filePath))
		}

		// Extract metadata from comments
		title, description, sql := parseRunBook(string(content))

		// Generate runBook ID from filename (without .sql extension)
		id := strings.TrimSuffix(entry.Name(), ".sql")

		runBooks[id] = &runBook{
			ID:          id,
			Title:       title,
			Description: description,
			FilePath:    filePath,
			Query:       sql,
		}
	}

	return runBooks, nil
}

// parseRunBook extracts title, description, and SQL from runBook content
func parseRunBook(content string) (title, description, sql string) {
	lines := strings.Split(content, "\n")
	var sqlLines []string
	titleFound := false
	descFound := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Check for title comment
		if strings.HasPrefix(trimmed, "-- title:") || strings.HasPrefix(trimmed, "--title:") {
			title = strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(trimmed, "-- title:"), "--title:"))
			titleFound = true
			continue
		}

		// Check for description comment
		if strings.HasPrefix(trimmed, "-- description:") || strings.HasPrefix(trimmed, "--description:") {
			description = strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(trimmed, "-- description:"), "--description:"))
			descFound = true
			continue
		}

		// Skip metadata lines
		if !titleFound && !descFound && strings.HasPrefix(trimmed, "--") {
			continue
		}

		// Add to SQL
		sqlLines = append(sqlLines, line)
	}

	sql = strings.TrimSpace(strings.Join(sqlLines, "\n"))
	return
}

// tableInfo represents a BigQuery table with metadata
type tableInfo struct {
	Project     string `yaml:"project"`
	Dataset     string `yaml:"dataset"`
	Table       string `yaml:"table"`
	Description string `yaml:"description"`
}

// FullName returns the full table name in the format project.dataset.table
func (t *tableInfo) FullName() string {
	return fmt.Sprintf("%s.%s.%s", t.Project, t.Dataset, t.Table)
}

// tableConfig represents the YAML configuration for tables
type tableConfig struct {
	Tables []tableInfo `yaml:"tables"`
}

// loadTableList loads a list of BigQuery tables from a YAML file
func loadTableList(filePath string) ([]tableInfo, error) {
	if filePath == "" {
		return nil, nil
	}

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil, goerr.Wrap(err, "table list file does not exist", goerr.V("file", filePath))
	}

	// Read file content
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to read table list file", goerr.V("file", filePath))
	}

	// Parse YAML
	var config tableConfig
	if err := yaml.Unmarshal(content, &config); err != nil {
		return nil, goerr.Wrap(err, "failed to parse YAML config", goerr.V("file", filePath))
	}

	return config.Tables, nil
}
