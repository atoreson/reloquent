package schema

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// LoadYAML reads a schema from a YAML file.
func LoadYAML(path string) (*Schema, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading schema file: %w", err)
	}
	s := &Schema{}
	if err := yaml.Unmarshal(data, s); err != nil {
		return nil, fmt.Errorf("parsing schema: %w", err)
	}
	return s, nil
}

// WriteYAML writes the schema to a YAML file at the given path.
func (s *Schema) WriteYAML(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("creating output directory: %w", err)
	}

	data, err := yaml.Marshal(s)
	if err != nil {
		return fmt.Errorf("marshaling schema: %w", err)
	}

	return os.WriteFile(path, data, 0o644)
}

// ToYAML returns the schema as a YAML byte slice.
func (s *Schema) ToYAML() ([]byte, error) {
	return yaml.Marshal(s)
}

// Summary returns a human-readable summary of the schema.
func (s *Schema) Summary() string {
	var totalRows int64
	var totalSize int64
	var totalCols int
	var totalFKs int

	for _, t := range s.Tables {
		totalRows += t.RowCount
		totalSize += t.SizeBytes
		totalCols += len(t.Columns)
		totalFKs += len(t.ForeignKeys)
	}

	return fmt.Sprintf(
		"Found %d tables, %d columns, %d foreign keys\nTotal rows: %d, Total size: %s",
		len(s.Tables), totalCols, totalFKs, totalRows, formatBytes(totalSize),
	)
}

func formatBytes(b int64) string {
	const (
		kb = 1024
		mb = kb * 1024
		gb = mb * 1024
		tb = gb * 1024
	)
	switch {
	case b >= tb:
		return fmt.Sprintf("%.1f TB", float64(b)/float64(tb))
	case b >= gb:
		return fmt.Sprintf("%.1f GB", float64(b)/float64(gb))
	case b >= mb:
		return fmt.Sprintf("%.1f MB", float64(b)/float64(mb))
	case b >= kb:
		return fmt.Sprintf("%.1f KB", float64(b)/float64(kb))
	default:
		return fmt.Sprintf("%d B", b)
	}
}
