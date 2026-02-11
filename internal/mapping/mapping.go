package mapping

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Mapping defines how source tables map to MongoDB collections.
type Mapping struct {
	Collections []Collection `yaml:"collections"`
}

// Collection represents a target MongoDB collection.
type Collection struct {
	Name            string           `yaml:"name"`
	SourceTable     string           `yaml:"source_table"`
	Embedded        []Embedded       `yaml:"embedded,omitempty"`
	References      []Reference      `yaml:"references,omitempty"`
	Transformations []Transformation `yaml:"transformations,omitempty"`
}

// Embedded represents a table whose rows are embedded as subdocuments.
type Embedded struct {
	SourceTable     string           `yaml:"source_table"`
	FieldName       string           `yaml:"field_name"`
	Relationship    string           `yaml:"relationship"` // array or single
	JoinColumn      string           `yaml:"join_column"`
	ParentColumn    string           `yaml:"parent_column"`
	Embedded        []Embedded       `yaml:"embedded,omitempty"`        // recursive nesting
	Transformations []Transformation `yaml:"transformations,omitempty"` // per-embedded transforms
}

// Reference represents a table kept as a separate collection, linked by a field.
type Reference struct {
	SourceTable  string `yaml:"source_table"`
	FieldName    string `yaml:"field_name"`
	JoinColumn   string `yaml:"join_column"`
	ParentColumn string `yaml:"parent_column"`
}

// WriteYAML writes the mapping to a YAML file at the given path.
func (m *Mapping) WriteYAML(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("creating output directory: %w", err)
	}

	data, err := yaml.Marshal(m)
	if err != nil {
		return fmt.Errorf("marshaling mapping: %w", err)
	}

	return os.WriteFile(path, data, 0o644)
}

// LoadYAML reads a mapping from a YAML file.
func LoadYAML(path string) (*Mapping, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading mapping file: %w", err)
	}
	m := &Mapping{}
	if err := yaml.Unmarshal(data, m); err != nil {
		return nil, fmt.Errorf("parsing mapping: %w", err)
	}
	return m, nil
}

// Transformation defines a per-field transformation rule.
type Transformation struct {
	SourceField string `yaml:"source_field"`
	Operation   string `yaml:"operation"` // rename, compute, cast, filter, default, exclude
	Value       string `yaml:"value,omitempty"`
	TargetField string `yaml:"target_field,omitempty"`
	TargetType  string `yaml:"target_type,omitempty"`
	Expression  string `yaml:"expression,omitempty"`
}
