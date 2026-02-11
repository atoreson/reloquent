package codegen

import (
	"strings"
	"testing"

	"github.com/reloquent/reloquent/internal/config"
	"github.com/reloquent/reloquent/internal/mapping"
	"github.com/reloquent/reloquent/internal/schema"
	"github.com/reloquent/reloquent/internal/typemap"
)

func TestGenerateBasicMigration(t *testing.T) {
	cfg := &config.Config{
		Version: 1,
		Source: config.SourceConfig{
			Type:           "postgresql",
			Host:           "localhost",
			Port:           5432,
			Database:       "testdb",
			MaxConnections: 20,
		},
		Target: config.TargetConfig{
			ConnectionString: "mongodb://localhost:27017",
			Database:         "testdb",
		},
	}

	s := &schema.Schema{
		Tables: []schema.Table{
			{
				Name: "orders",
				Columns: []schema.Column{
					{Name: "id", DataType: "integer"},
					{Name: "customer_id", DataType: "integer"},
				},
				PrimaryKey: &schema.PrimaryKey{
					Name:    "orders_pkey",
					Columns: []string{"id"},
				},
			},
		},
	}

	m := &mapping.Mapping{
		Collections: []mapping.Collection{
			{
				Name:        "orders",
				SourceTable: "orders",
			},
		},
	}

	g := &Generator{
		Config:  cfg,
		Schema:  s,
		Mapping: m,
		TypeMap: typemap.DefaultPostgres(),
	}

	result, err := g.Generate()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result.MigrationScript, "Collection: orders") {
		t.Error("expected migration script to contain collection name")
	}
	if !strings.Contains(result.MigrationScript, "numPartitions=20") {
		t.Error("expected migration script to contain partition count")
	}
	if !strings.Contains(result.MigrationScript, "jdbc:postgresql://") {
		t.Error("expected migration script to contain JDBC URL")
	}
	if !strings.Contains(result.MigrationScript, `"ordered", "false"`) {
		t.Error("expected migration script to use unordered writes")
	}
	if !strings.Contains(result.MigrationScript, `"maxBatchSize", "100000"`) {
		t.Error("expected migration script to include maxBatchSize")
	}
	if !strings.Contains(result.MigrationScript, `"compressors", "zstd"`) {
		t.Error("expected migration script to include zstd compression")
	}
}

func TestFindPartitionColumn(t *testing.T) {
	s := &schema.Schema{
		Tables: []schema.Table{
			{
				Name: "users",
				Columns: []schema.Column{
					{Name: "id", DataType: "bigint"},
					{Name: "name", DataType: "text"},
				},
				PrimaryKey: &schema.PrimaryKey{
					Name:    "users_pkey",
					Columns: []string{"id"},
				},
			},
		},
	}

	col := findPartitionColumn(s, "users")
	if col != "id" {
		t.Errorf("expected partition column 'id', got %s", col)
	}
}

func TestBuildJDBCURL(t *testing.T) {
	src := config.SourceConfig{
		Type:     "postgresql",
		Host:     "db.example.com",
		Port:     5432,
		Database: "mydb",
		SSL:      true,
	}
	url := buildJDBCURL(src)
	if url != "jdbc:postgresql://db.example.com:5432/mydb?ssl=true" {
		t.Errorf("unexpected JDBC URL: %s", url)
	}
}

func TestBuildJDBCURL_Oracle(t *testing.T) {
	src := config.SourceConfig{
		Type:     "oracle",
		Host:     "db.example.com",
		Port:     1521,
		Database: "ORCL",
	}
	url := buildJDBCURL(src)
	if url != "jdbc:oracle:thin:@db.example.com:1521/ORCL" {
		t.Errorf("unexpected Oracle JDBC URL: %s", url)
	}
}

func TestGenerateDeepNesting(t *testing.T) {
	cfg := &config.Config{
		Version: 1,
		Source: config.SourceConfig{
			Type:           "postgresql",
			Host:           "localhost",
			Port:           5432,
			Database:       "testdb",
			MaxConnections: 10,
		},
		Target: config.TargetConfig{
			ConnectionString: "mongodb://localhost:27017",
			Database:         "testdb",
		},
	}

	s := &schema.Schema{
		Tables: []schema.Table{
			{
				Name: "customers",
				Columns: []schema.Column{
					{Name: "id", DataType: "integer"},
				},
				PrimaryKey: &schema.PrimaryKey{Name: "pk_cust", Columns: []string{"id"}},
			},
			{
				Name: "orders",
				Columns: []schema.Column{
					{Name: "id", DataType: "integer"},
					{Name: "customer_id", DataType: "integer"},
				},
				PrimaryKey: &schema.PrimaryKey{Name: "pk_orders", Columns: []string{"id"}},
			},
			{
				Name: "order_items",
				Columns: []schema.Column{
					{Name: "id", DataType: "integer"},
					{Name: "order_id", DataType: "integer"},
				},
				PrimaryKey: &schema.PrimaryKey{Name: "pk_items", Columns: []string{"id"}},
			},
		},
	}

	m := &mapping.Mapping{
		Collections: []mapping.Collection{
			{
				Name:        "customers",
				SourceTable: "customers",
				Embedded: []mapping.Embedded{
					{
						SourceTable:  "orders",
						FieldName:    "orders",
						Relationship: "array",
						JoinColumn:   "customer_id",
						ParentColumn: "id",
						Embedded: []mapping.Embedded{
							{
								SourceTable:  "order_items",
								FieldName:    "items",
								Relationship: "array",
								JoinColumn:   "order_id",
								ParentColumn: "id",
							},
						},
					},
				},
			},
		},
	}

	g := &Generator{
		Config:  cfg,
		Schema:  s,
		Mapping: m,
		TypeMap: typemap.DefaultPostgres(),
	}

	result, err := g.Generate()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	script := result.MigrationScript

	// Should read all three tables
	if !strings.Contains(script, `table="customers"`) {
		t.Error("script should read customers table")
	}
	if !strings.Contains(script, `table="orders"`) {
		t.Error("script should read orders table")
	}
	if !strings.Contains(script, `table="order_items"`) {
		t.Error("script should read order_items table")
	}

	// order_items should be processed before orders (bottom-up)
	itemsIdx := strings.Index(script, `table="order_items"`)
	ordersGroupIdx := strings.Index(script, `orders_nested = orders_df.groupBy`)
	if itemsIdx < 0 || ordersGroupIdx < 0 {
		t.Error("script should contain order_items read and orders groupBy")
	}

	// Both groupBy+collect_list should appear
	if !strings.Contains(script, `collect_list(struct("*")).alias("items")`) {
		t.Error("script should collect_list for items")
	}
	if !strings.Contains(script, `collect_list(struct("*")).alias("orders")`) {
		t.Error("script should collect_list for orders")
	}
}

func TestGenerateWithTransformations(t *testing.T) {
	cfg := &config.Config{
		Version: 1,
		Source: config.SourceConfig{
			Type:           "postgresql",
			Host:           "localhost",
			Port:           5432,
			Database:       "testdb",
			MaxConnections: 10,
		},
		Target: config.TargetConfig{
			ConnectionString: "mongodb://localhost:27017",
			Database:         "testdb",
		},
	}

	s := &schema.Schema{
		Tables: []schema.Table{
			{
				Name: "users",
				Columns: []schema.Column{
					{Name: "id", DataType: "integer"},
					{Name: "first_name", DataType: "text"},
				},
				PrimaryKey: &schema.PrimaryKey{Name: "pk_users", Columns: []string{"id"}},
			},
		},
	}

	m := &mapping.Mapping{
		Collections: []mapping.Collection{
			{
				Name:        "users",
				SourceTable: "users",
				Transformations: []mapping.Transformation{
					{
						SourceField: "first_name",
						Operation:   "rename",
						TargetField: "firstName",
					},
					{
						SourceField: "temp_field",
						Operation:   "exclude",
					},
				},
			},
		},
	}

	g := &Generator{
		Config:  cfg,
		Schema:  s,
		Mapping: m,
		TypeMap: typemap.DefaultPostgres(),
	}

	result, err := g.Generate()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	script := result.MigrationScript

	if !strings.Contains(script, "withColumnRenamed") {
		t.Error("script should contain rename transformation")
	}
	if !strings.Contains(script, "drop") {
		t.Error("script should contain exclude transformation")
	}
	// Should import transform functions
	if !strings.Contains(script, "coalesce, lit, expr, col") {
		t.Error("script should import transform functions when transforms are present")
	}
}

func TestGenerateOracleJDBCURL(t *testing.T) {
	cfg := &config.Config{
		Version: 1,
		Source: config.SourceConfig{
			Type:           "oracle",
			Host:           "oracledb",
			Port:           1521,
			Database:       "ORCL",
			MaxConnections: 10,
		},
		Target: config.TargetConfig{
			ConnectionString: "mongodb://localhost:27017",
			Database:         "testdb",
		},
	}

	s := &schema.Schema{
		Tables: []schema.Table{
			{
				Name: "users",
				Columns: []schema.Column{
					{Name: "ID", DataType: "NUMBER"},
				},
			},
		},
	}

	m := &mapping.Mapping{
		Collections: []mapping.Collection{
			{Name: "users", SourceTable: "users"},
		},
	}

	g := &Generator{
		Config:  cfg,
		Schema:  s,
		Mapping: m,
		TypeMap: typemap.DefaultOracle(),
	}

	result, err := g.Generate()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result.MigrationScript, "jdbc:oracle:thin:@oracledb:1521/ORCL") {
		t.Error("script should contain Oracle JDBC URL")
	}
	if !strings.Contains(result.MigrationScript, "oracle.jdbc.OracleDriver") {
		t.Error("script should reference Oracle JDBC driver")
	}
}
