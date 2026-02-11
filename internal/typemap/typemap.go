package typemap

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"gopkg.in/yaml.v3"
)

// BSONType represents a MongoDB BSON type.
type BSONType string

const (
	BSONNumberLong BSONType = "NumberLong"
	BSONDecimal128 BSONType = "Decimal128"
	BSONString     BSONType = "String"
	BSONISODate    BSONType = "ISODate"
	BSONBinData    BSONType = "BinData"
	BSONDocument   BSONType = "Document"
	BSONArray      BSONType = "Array"
	BSONBoolean    BSONType = "Boolean"
	BSONDouble     BSONType = "Double"
)

// AllBSONTypes lists all known BSON types for cycling in the editor.
var AllBSONTypes = []BSONType{
	BSONNumberLong,
	BSONDecimal128,
	BSONString,
	BSONISODate,
	BSONBinData,
	BSONDocument,
	BSONArray,
	BSONBoolean,
	BSONDouble,
}

// TypeMap holds the mapping from source types to BSON types.
type TypeMap struct {
	Mappings  map[string]BSONType `yaml:"mappings"`
	Overrides map[string]BSONType `yaml:"overrides,omitempty"`
	defaults  map[string]BSONType // not serialized; populated by ForDatabase
}

// DefaultPostgres returns the default type mapping for PostgreSQL.
func DefaultPostgres() *TypeMap {
	m := map[string]BSONType{
		"integer":                     BSONNumberLong,
		"bigint":                      BSONNumberLong,
		"smallint":                    BSONNumberLong,
		"serial":                      BSONNumberLong,
		"bigserial":                   BSONNumberLong,
		"numeric":                     BSONDecimal128,
		"decimal":                     BSONDecimal128,
		"real":                        BSONDouble,
		"double precision":            BSONDouble,
		"character varying":           BSONString,
		"varchar":                     BSONString,
		"text":                        BSONString,
		"char":                        BSONString,
		"character":                   BSONString,
		"boolean":                     BSONBoolean,
		"date":                        BSONISODate,
		"timestamp":                   BSONISODate,
		"timestamp with time zone":    BSONISODate,
		"timestamp without time zone": BSONISODate,
		"bytea":                       BSONBinData,
		"uuid":                        BSONString,
		"jsonb":                       BSONDocument,
		"json":                        BSONDocument,
		"ARRAY":                       BSONArray,
	}
	return &TypeMap{Mappings: m}
}

// DefaultOracle returns the default type mapping for Oracle.
func DefaultOracle() *TypeMap {
	m := map[string]BSONType{
		"NUMBER":    BSONNumberLong,
		"VARCHAR2":  BSONString,
		"NVARCHAR2": BSONString,
		"CHAR":      BSONString,
		"NCHAR":     BSONString,
		"CLOB":      BSONString,
		"NCLOB":     BSONString,
		"DATE":      BSONISODate,
		"TIMESTAMP": BSONISODate,
		"BLOB":      BSONBinData,
		"RAW":       BSONString,
	}
	return &TypeMap{Mappings: m}
}

// ForDatabase returns a TypeMap with defaults for the given database type.
func ForDatabase(dbType string) *TypeMap {
	var tm *TypeMap
	switch dbType {
	case "oracle":
		tm = DefaultOracle()
	default:
		tm = DefaultPostgres()
	}
	// Store defaults for override tracking
	tm.defaults = make(map[string]BSONType, len(tm.Mappings))
	for k, v := range tm.Mappings {
		tm.defaults[k] = v
	}
	if tm.Overrides == nil {
		tm.Overrides = make(map[string]BSONType)
	}
	return tm
}

// Resolve returns the BSON type for the given source type.
func (tm *TypeMap) Resolve(sourceType string) BSONType {
	if bsonType, ok := tm.Mappings[sourceType]; ok {
		return bsonType
	}
	return BSONString // fallback
}

// Override applies a user override for a source type.
func (tm *TypeMap) Override(sourceType string, bsonType BSONType) {
	tm.Mappings[sourceType] = bsonType
	if tm.Overrides == nil {
		tm.Overrides = make(map[string]BSONType)
	}
	// Track override only if different from default
	if tm.defaults != nil {
		if def, ok := tm.defaults[sourceType]; ok && def == bsonType {
			delete(tm.Overrides, sourceType)
			return
		}
	}
	tm.Overrides[sourceType] = bsonType
}

// RestoreDefault restores the default mapping for a source type.
func (tm *TypeMap) RestoreDefault(sourceType string) {
	if tm.defaults != nil {
		if def, ok := tm.defaults[sourceType]; ok {
			tm.Mappings[sourceType] = def
			delete(tm.Overrides, sourceType)
		}
	}
}

// IsOverridden returns true if the source type has been overridden from its default.
func (tm *TypeMap) IsOverridden(sourceType string) bool {
	if tm.Overrides == nil {
		return false
	}
	_, ok := tm.Overrides[sourceType]
	return ok
}

// AllMappings returns all source type -> BSON type mappings, sorted by key.
func (tm *TypeMap) AllMappings() map[string]BSONType {
	return tm.Mappings
}

// SortedTypes returns the source type names sorted alphabetically.
func (tm *TypeMap) SortedTypes() []string {
	types := make([]string, 0, len(tm.Mappings))
	for k := range tm.Mappings {
		types = append(types, k)
	}
	sort.Strings(types)
	return types
}

// WriteYAML writes the type mapping to a YAML file.
func (tm *TypeMap) WriteYAML(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("creating output directory: %w", err)
	}

	data, err := yaml.Marshal(tm)
	if err != nil {
		return fmt.Errorf("marshaling type map: %w", err)
	}

	return os.WriteFile(path, data, 0o644)
}

// LoadYAML reads a type mapping from a YAML file.
func LoadYAML(path string) (*TypeMap, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading type map file: %w", err)
	}
	tm := &TypeMap{}
	if err := yaml.Unmarshal(data, tm); err != nil {
		return nil, fmt.Errorf("parsing type map: %w", err)
	}
	if tm.Mappings == nil {
		tm.Mappings = make(map[string]BSONType)
	}
	if tm.Overrides == nil {
		tm.Overrides = make(map[string]BSONType)
	}
	return tm, nil
}
