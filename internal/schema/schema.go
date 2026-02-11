package schema

// Schema represents the complete discovered schema of a source database.
type Schema struct {
	DatabaseType string  `yaml:"database_type"` // postgresql or oracle
	Host         string  `yaml:"host"`
	Database     string  `yaml:"database"`
	SchemaName   string  `yaml:"schema_name,omitempty"`
	Tables       []Table `yaml:"tables"`
}

// Table represents a database table.
type Table struct {
	Name        string       `yaml:"name"`
	Columns     []Column     `yaml:"columns"`
	PrimaryKey  *PrimaryKey  `yaml:"primary_key,omitempty"`
	ForeignKeys []ForeignKey `yaml:"foreign_keys,omitempty"`
	Indexes     []Index      `yaml:"indexes,omitempty"`
	Constraints []Constraint `yaml:"constraints,omitempty"`
	RowCount    int64        `yaml:"row_count"`
	SizeBytes   int64        `yaml:"size_bytes"`
}

// Column represents a table column.
type Column struct {
	Name         string  `yaml:"name"`
	DataType     string  `yaml:"data_type"`
	Nullable     bool    `yaml:"nullable"`
	DefaultValue *string `yaml:"default_value,omitempty"`
	MaxLength    *int    `yaml:"max_length,omitempty"`
	Precision    *int    `yaml:"precision,omitempty"`
	Scale        *int    `yaml:"scale,omitempty"`
	IsSequence   bool    `yaml:"is_sequence,omitempty"`
}

// PrimaryKey represents a table's primary key.
type PrimaryKey struct {
	Name    string   `yaml:"name"`
	Columns []string `yaml:"columns"`
}

// ForeignKey represents a foreign key relationship.
type ForeignKey struct {
	Name              string   `yaml:"name"`
	Columns           []string `yaml:"columns"`
	ReferencedTable   string   `yaml:"referenced_table"`
	ReferencedColumns []string `yaml:"referenced_columns"`
}

// Index represents a database index.
type Index struct {
	Name    string   `yaml:"name"`
	Columns []string `yaml:"columns"`
	Unique  bool     `yaml:"unique"`
	Type    string   `yaml:"type,omitempty"` // btree, hash, gin, gist, etc.
}

// Constraint represents a check constraint or enum.
type Constraint struct {
	Name       string `yaml:"name"`
	Type       string `yaml:"type"` // check, enum
	Definition string `yaml:"definition"`
}
