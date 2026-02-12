package schema

// Schema represents the complete discovered schema of a source database.
type Schema struct {
	DatabaseType string  `yaml:"database_type" json:"database_type"`
	Host         string  `yaml:"host" json:"host"`
	Database     string  `yaml:"database" json:"database"`
	SchemaName   string  `yaml:"schema_name,omitempty" json:"schema_name,omitempty"`
	Tables       []Table `yaml:"tables" json:"tables"`
}

// Table represents a database table.
type Table struct {
	Name        string       `yaml:"name" json:"name"`
	Columns     []Column     `yaml:"columns" json:"columns"`
	PrimaryKey  *PrimaryKey  `yaml:"primary_key,omitempty" json:"primary_key,omitempty"`
	ForeignKeys []ForeignKey `yaml:"foreign_keys,omitempty" json:"foreign_keys,omitempty"`
	Indexes     []Index      `yaml:"indexes,omitempty" json:"indexes,omitempty"`
	Constraints []Constraint `yaml:"constraints,omitempty" json:"constraints,omitempty"`
	RowCount    int64        `yaml:"row_count" json:"row_count"`
	SizeBytes   int64        `yaml:"size_bytes" json:"size_bytes"`
}

// Column represents a table column.
type Column struct {
	Name         string  `yaml:"name" json:"name"`
	DataType     string  `yaml:"data_type" json:"data_type"`
	Nullable     bool    `yaml:"nullable" json:"nullable"`
	DefaultValue *string `yaml:"default_value,omitempty" json:"default_value,omitempty"`
	MaxLength    *int    `yaml:"max_length,omitempty" json:"max_length,omitempty"`
	Precision    *int    `yaml:"precision,omitempty" json:"precision,omitempty"`
	Scale        *int    `yaml:"scale,omitempty" json:"scale,omitempty"`
	IsSequence   bool    `yaml:"is_sequence,omitempty" json:"is_sequence,omitempty"`
}

// PrimaryKey represents a table's primary key.
type PrimaryKey struct {
	Name    string   `yaml:"name" json:"name"`
	Columns []string `yaml:"columns" json:"columns"`
}

// ForeignKey represents a foreign key relationship.
type ForeignKey struct {
	Name              string   `yaml:"name" json:"name"`
	Columns           []string `yaml:"columns" json:"columns"`
	ReferencedTable   string   `yaml:"referenced_table" json:"referenced_table"`
	ReferencedColumns []string `yaml:"referenced_columns" json:"referenced_columns"`
}

// Index represents a database index.
type Index struct {
	Name    string   `yaml:"name" json:"name"`
	Columns []string `yaml:"columns" json:"columns"`
	Unique  bool     `yaml:"unique" json:"unique"`
	Type    string   `yaml:"type,omitempty" json:"type,omitempty"`
}

// Constraint represents a check constraint or enum.
type Constraint struct {
	Name       string `yaml:"name" json:"name"`
	Type       string `yaml:"type" json:"type"`
	Definition string `yaml:"definition" json:"definition"`
}
