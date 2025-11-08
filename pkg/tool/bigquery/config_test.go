package bigquery

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/m-mizutani/gt"
)

func TestLoadRunBooks(t *testing.T) {
	// Create temporary directory with test runbooks
	tmpDir := t.TempDir()

	// Create test runbook 1
	runbook1 := `-- title: Test Query 1
-- description: A test query for testing

SELECT 1 as test_value`

	err := os.WriteFile(filepath.Join(tmpDir, "test1.sql"), []byte(runbook1), 0644)
	gt.NoError(t, err)

	// Create test runbook 2 without metadata
	runbook2 := `SELECT 2 as another_value`

	err = os.WriteFile(filepath.Join(tmpDir, "test2.sql"), []byte(runbook2), 0644)
	gt.NoError(t, err)

	// Create non-SQL file (should be ignored)
	err = os.WriteFile(filepath.Join(tmpDir, "readme.txt"), []byte("readme"), 0644)
	gt.NoError(t, err)

	// Load runbooks
	runBooks, err := loadRunBooks(tmpDir)
	gt.NoError(t, err)
	gt.Equal(t, len(runBooks), 2)

	// Check test1
	rb1, exists := runBooks["test1"]
	gt.True(t, exists)
	gt.Equal(t, rb1.ID, "test1")
	gt.Equal(t, rb1.Title, "Test Query 1")
	gt.Equal(t, rb1.Description, "A test query for testing")
	gt.S(t, rb1.SQL).Contains("SELECT 1 as test_value")

	// Check test2
	rb2, exists := runBooks["test2"]
	gt.True(t, exists)
	gt.Equal(t, rb2.ID, "test2")
	gt.Equal(t, rb2.Title, "")
	gt.Equal(t, rb2.Description, "")
	gt.S(t, rb2.SQL).Contains("SELECT 2 as another_value")
}

func TestLoadRunBooks_EmptyDir(t *testing.T) {
	runBooks, err := loadRunBooks("")
	gt.NoError(t, err)
	gt.Nil(t, runBooks)
}

func TestLoadRunBooks_NonExistentDir(t *testing.T) {
	_, err := loadRunBooks("/nonexistent/directory")
	gt.Error(t, err).Required()
}

func TestParseRunBook(t *testing.T) {
	testCases := []struct {
		name        string
		content     string
		wantTitle   string
		wantDesc    string
		wantSQL     string
	}{
		{
			name: "with metadata",
			content: `-- title: My Query
-- description: This is a description

SELECT * FROM table`,
			wantTitle: "My Query",
			wantDesc:  "This is a description",
			wantSQL:   "SELECT * FROM table",
		},
		{
			name: "without metadata",
			content: `SELECT * FROM table`,
			wantTitle: "",
			wantDesc:  "",
			wantSQL:   "SELECT * FROM table",
		},
		{
			name: "with inline metadata",
			content: `--title:Inline Title
--description:Inline Description
SELECT 1`,
			wantTitle: "Inline Title",
			wantDesc:  "Inline Description",
			wantSQL:   "SELECT 1",
		},
		{
			name: "with comments before metadata",
			content: `-- This is a comment
-- title: Query
-- Another comment
-- description: Desc
SELECT 1`,
			wantTitle: "Query",
			wantDesc:  "Desc",
			wantSQL:   "SELECT 1",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			title, desc, sql := parseRunBook(tc.content)
			gt.Equal(t, title, tc.wantTitle)
			gt.Equal(t, desc, tc.wantDesc)
			gt.S(t, sql).Contains(tc.wantSQL)
		})
	}
}

func TestLoadTableList(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "config.yaml")

	// Create test config
	configContent := `tables:
  - project: proj1
    dataset: ds1
    table: tbl1
    description: Test table 1
  - project: proj2
    dataset: ds2
    table: tbl2
    description: Test table 2
  - project: proj3
    dataset: ds3
    table: tbl3`

	err := os.WriteFile(configFile, []byte(configContent), 0644)
	gt.NoError(t, err)

	// Load table list
	tables, err := loadTableList(configFile)
	gt.NoError(t, err)
	gt.Equal(t, len(tables), 3)

	// Check first table
	gt.Equal(t, tables[0].Project, "proj1")
	gt.Equal(t, tables[0].Dataset, "ds1")
	gt.Equal(t, tables[0].Table, "tbl1")
	gt.Equal(t, tables[0].Description, "Test table 1")
	gt.Equal(t, tables[0].FullName(), "proj1.ds1.tbl1")

	// Check second table
	gt.Equal(t, tables[1].Project, "proj2")
	gt.Equal(t, tables[1].Dataset, "ds2")
	gt.Equal(t, tables[1].Table, "tbl2")
	gt.Equal(t, tables[1].Description, "Test table 2")
	gt.Equal(t, tables[1].FullName(), "proj2.ds2.tbl2")

	// Check third table (no description)
	gt.Equal(t, tables[2].Project, "proj3")
	gt.Equal(t, tables[2].Dataset, "ds3")
	gt.Equal(t, tables[2].Table, "tbl3")
	gt.Equal(t, tables[2].Description, "")
	gt.Equal(t, tables[2].FullName(), "proj3.ds3.tbl3")
}

func TestLoadTableList_EmptyPath(t *testing.T) {
	tables, err := loadTableList("")
	gt.NoError(t, err)
	gt.Nil(t, tables)
}

func TestLoadTableList_NonExistentFile(t *testing.T) {
	_, err := loadTableList("/nonexistent/config.yaml")
	gt.Error(t, err).Required()
}

func TestLoadTableList_InvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "invalid.yaml")

	// Create invalid YAML
	err := os.WriteFile(configFile, []byte("invalid: yaml: content:"), 0644)
	gt.NoError(t, err)

	_, err = loadTableList(configFile)
	gt.Error(t, err).Required()
}

func TestTableInfoFullName(t *testing.T) {
	table := tableInfo{
		Project: "my-project",
		Dataset: "my_dataset",
		Table:   "my_table",
	}

	gt.Equal(t, table.FullName(), "my-project.my_dataset.my_table")
}
