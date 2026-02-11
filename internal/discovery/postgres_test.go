package discovery_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/reloquent/reloquent/internal/config"
	"github.com/reloquent/reloquent/internal/discovery"
)

// pgTestConfig returns a SourceConfig from environment variables.
// Set RELOQUENT_TEST_PG_HOST (default localhost), RELOQUENT_TEST_PG_PORT (default 5432),
// RELOQUENT_TEST_PG_DATABASE (default reloquent_test), RELOQUENT_TEST_PG_USER (default postgres),
// RELOQUENT_TEST_PG_PASSWORD (default postgres) to configure.
func pgTestConfig() *config.SourceConfig {
	host := os.Getenv("RELOQUENT_TEST_PG_HOST")
	if host == "" {
		host = "localhost"
	}
	port := 5432
	db := os.Getenv("RELOQUENT_TEST_PG_DATABASE")
	if db == "" {
		db = "reloquent_test"
	}
	user := os.Getenv("RELOQUENT_TEST_PG_USER")
	if user == "" {
		user = "postgres"
	}
	pass := os.Getenv("RELOQUENT_TEST_PG_PASSWORD")
	if pass == "" {
		pass = "postgres"
	}
	return &config.SourceConfig{
		Type:     "postgresql",
		Host:     host,
		Port:     port,
		Database: db,
		Username: user,
		Password: pass,
		Schema:   "public",
	}
}

// skipIfNoPostgres skips the test if a PostgreSQL test instance is not available.
func skipIfNoPostgres(t *testing.T, cfg *config.SourceConfig) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	connStr := fmt.Sprintf("host=%s port=%d dbname=%s user=%s password=%s sslmode=disable",
		cfg.Host, cfg.Port, cfg.Database, cfg.Username, cfg.Password)

	pool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		t.Skipf("skipping: cannot connect to PostgreSQL: %v", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		t.Skipf("skipping: cannot ping PostgreSQL: %v", err)
	}
	pool.Close()
}

// setupTestSchema creates a test schema with tables, columns, PKs, FKs, indexes, constraints.
func setupTestSchema(t *testing.T, cfg *config.SourceConfig) func() {
	t.Helper()
	ctx := context.Background()

	connStr := fmt.Sprintf("host=%s port=%d dbname=%s user=%s password=%s sslmode=disable",
		cfg.Host, cfg.Port, cfg.Database, cfg.Username, cfg.Password)
	pool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		t.Fatalf("connect for setup: %v", err)
	}

	ddl := []string{
		`DROP TABLE IF EXISTS order_items CASCADE`,
		`DROP TABLE IF EXISTS orders CASCADE`,
		`DROP TABLE IF EXISTS customers CASCADE`,
		`CREATE TABLE customers (
			id SERIAL PRIMARY KEY,
			email VARCHAR(255) NOT NULL UNIQUE,
			name TEXT NOT NULL,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT now(),
			score NUMERIC(10,2),
			CONSTRAINT customers_score_positive CHECK (score >= 0)
		)`,
		`CREATE TABLE orders (
			id BIGSERIAL PRIMARY KEY,
			customer_id INTEGER NOT NULL REFERENCES customers(id),
			order_date DATE NOT NULL,
			total NUMERIC(12,2) NOT NULL,
			status VARCHAR(20) DEFAULT 'pending'
		)`,
		`CREATE INDEX idx_orders_customer_id ON orders(customer_id)`,
		`CREATE INDEX idx_orders_date_status ON orders(order_date, status)`,
		`CREATE TABLE order_items (
			id BIGSERIAL PRIMARY KEY,
			order_id BIGINT NOT NULL REFERENCES orders(id),
			product_name TEXT NOT NULL,
			quantity INTEGER NOT NULL DEFAULT 1,
			unit_price NUMERIC(10,2) NOT NULL,
			CONSTRAINT order_items_qty_positive CHECK (quantity > 0)
		)`,
		`CREATE INDEX idx_order_items_order_id ON order_items(order_id)`,
		// Insert some test data so row counts are non-zero after ANALYZE
		`INSERT INTO customers (email, name, score) VALUES
			('alice@example.com', 'Alice', 100.50),
			('bob@example.com', 'Bob', 200.00),
			('carol@example.com', 'Carol', NULL)`,
		`INSERT INTO orders (customer_id, order_date, total) VALUES
			(1, '2024-01-15', 99.99),
			(1, '2024-02-20', 249.50),
			(2, '2024-03-10', 50.00)`,
		`INSERT INTO order_items (order_id, product_name, quantity, unit_price) VALUES
			(1, 'Widget', 2, 25.00),
			(1, 'Gadget', 1, 49.99),
			(2, 'Widget', 5, 25.00),
			(2, 'Gizmo', 1, 124.50),
			(3, 'Gadget', 1, 50.00)`,
		`ANALYZE customers`,
		`ANALYZE orders`,
		`ANALYZE order_items`,
	}

	for _, stmt := range ddl {
		if _, err := pool.Exec(ctx, stmt); err != nil {
			pool.Close()
			t.Fatalf("setup DDL failed: %s: %v", stmt, err)
		}
	}
	pool.Close()

	return func() {
		pool2, err := pgxpool.New(ctx, connStr)
		if err != nil {
			return
		}
		defer pool2.Close()
		pool2.Exec(ctx, "DROP TABLE IF EXISTS order_items CASCADE")
		pool2.Exec(ctx, "DROP TABLE IF EXISTS orders CASCADE")
		pool2.Exec(ctx, "DROP TABLE IF EXISTS customers CASCADE")
	}
}

func TestPostgresDiscoverIntegration(t *testing.T) {
	cfg := pgTestConfig()
	skipIfNoPostgres(t, cfg)

	cleanup := setupTestSchema(t, cfg)
	defer cleanup()

	ctx := context.Background()

	d, err := discovery.NewPostgres(cfg)
	if err != nil {
		t.Fatalf("NewPostgres: %v", err)
	}
	defer d.Close()

	if err := d.Connect(ctx); err != nil {
		t.Fatalf("Connect: %v", err)
	}

	s, err := d.Discover(ctx)
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}

	// Verify schema metadata
	if s.DatabaseType != "postgresql" {
		t.Errorf("expected database_type postgresql, got %s", s.DatabaseType)
	}

	// Should find our 3 test tables
	if len(s.Tables) < 3 {
		t.Fatalf("expected at least 3 tables, got %d", len(s.Tables))
	}

	tableByName := make(map[string]int)
	for i, tbl := range s.Tables {
		tableByName[tbl.Name] = i
	}

	// --- customers table ---
	t.Run("customers", func(t *testing.T) {
		idx, ok := tableByName["customers"]
		if !ok {
			t.Fatal("customers table not found")
		}
		tbl := s.Tables[idx]

		// Columns
		if len(tbl.Columns) != 5 {
			t.Errorf("expected 5 columns, got %d", len(tbl.Columns))
		}

		colByName := make(map[string]*int)
		for i := range tbl.Columns {
			colByName[tbl.Columns[i].Name] = &i
		}

		// Check id column is serial (sequence)
		if idIdx, ok := colByName["id"]; ok {
			col := tbl.Columns[*idIdx]
			if !col.IsSequence {
				t.Error("expected id column to be marked as sequence")
			}
			if col.DataType != "integer" {
				t.Errorf("expected id data_type integer, got %s", col.DataType)
			}
		} else {
			t.Error("id column not found")
		}

		// Check email column
		if emailIdx, ok := colByName["email"]; ok {
			col := tbl.Columns[*emailIdx]
			if col.Nullable {
				t.Error("expected email to be NOT NULL")
			}
			if col.DataType != "character varying" {
				t.Errorf("expected email data_type character varying, got %s", col.DataType)
			}
			if col.MaxLength == nil || *col.MaxLength != 255 {
				t.Error("expected email max_length 255")
			}
		} else {
			t.Error("email column not found")
		}

		// Check score column
		if scoreIdx, ok := colByName["score"]; ok {
			col := tbl.Columns[*scoreIdx]
			if !col.Nullable {
				t.Error("expected score to be nullable")
			}
			if col.DataType != "numeric" {
				t.Errorf("expected score data_type numeric, got %s", col.DataType)
			}
			if col.Precision == nil || *col.Precision != 10 {
				t.Error("expected score precision 10")
			}
			if col.Scale == nil || *col.Scale != 2 {
				t.Error("expected score scale 2")
			}
		} else {
			t.Error("score column not found")
		}

		// Primary key
		if tbl.PrimaryKey == nil {
			t.Fatal("expected primary key")
		}
		if len(tbl.PrimaryKey.Columns) != 1 || tbl.PrimaryKey.Columns[0] != "id" {
			t.Errorf("expected PK on (id), got %v", tbl.PrimaryKey.Columns)
		}

		// Row count (should be 3 after ANALYZE)
		if tbl.RowCount != 3 {
			t.Errorf("expected row count 3, got %d", tbl.RowCount)
		}

		// Size should be > 0
		if tbl.SizeBytes <= 0 {
			t.Error("expected positive size_bytes")
		}

		// Check constraint
		if len(tbl.Constraints) == 0 {
			t.Error("expected at least one check constraint (score >= 0)")
		}
		foundScoreCheck := false
		for _, c := range tbl.Constraints {
			if c.Type == "check" && c.Name == "customers_score_positive" {
				foundScoreCheck = true
			}
		}
		if !foundScoreCheck {
			t.Error("expected customers_score_positive check constraint")
		}

		// Unique index on email
		foundEmailIdx := false
		for _, idx := range tbl.Indexes {
			for _, col := range idx.Columns {
				if col == "email" && idx.Unique {
					foundEmailIdx = true
				}
			}
		}
		if !foundEmailIdx {
			t.Error("expected unique index on email")
		}
	})

	// --- orders table ---
	t.Run("orders", func(t *testing.T) {
		idx, ok := tableByName["orders"]
		if !ok {
			t.Fatal("orders table not found")
		}
		tbl := s.Tables[idx]

		// Foreign key to customers
		if len(tbl.ForeignKeys) != 1 {
			t.Fatalf("expected 1 foreign key, got %d", len(tbl.ForeignKeys))
		}
		fk := tbl.ForeignKeys[0]
		if fk.ReferencedTable != "customers" {
			t.Errorf("expected FK to customers, got %s", fk.ReferencedTable)
		}
		if len(fk.Columns) != 1 || fk.Columns[0] != "customer_id" {
			t.Errorf("expected FK column customer_id, got %v", fk.Columns)
		}

		// Indexes: idx_orders_customer_id and idx_orders_date_status
		if len(tbl.Indexes) < 2 {
			t.Errorf("expected at least 2 indexes, got %d", len(tbl.Indexes))
		}

		foundComposite := false
		for _, idx := range tbl.Indexes {
			if idx.Name == "idx_orders_date_status" {
				foundComposite = true
				if len(idx.Columns) != 2 {
					t.Errorf("expected composite index with 2 columns, got %d", len(idx.Columns))
				}
			}
		}
		if !foundComposite {
			t.Error("expected idx_orders_date_status composite index")
		}

		// Row count
		if tbl.RowCount != 3 {
			t.Errorf("expected row count 3, got %d", tbl.RowCount)
		}
	})

	// --- order_items table ---
	t.Run("order_items", func(t *testing.T) {
		idx, ok := tableByName["order_items"]
		if !ok {
			t.Fatal("order_items table not found")
		}
		tbl := s.Tables[idx]

		// Foreign key to orders
		if len(tbl.ForeignKeys) != 1 {
			t.Fatalf("expected 1 foreign key, got %d", len(tbl.ForeignKeys))
		}
		if tbl.ForeignKeys[0].ReferencedTable != "orders" {
			t.Errorf("expected FK to orders, got %s", tbl.ForeignKeys[0].ReferencedTable)
		}

		// Check constraint
		foundQtyCheck := false
		for _, c := range tbl.Constraints {
			if c.Name == "order_items_qty_positive" {
				foundQtyCheck = true
			}
		}
		if !foundQtyCheck {
			t.Error("expected order_items_qty_positive check constraint")
		}

		// Row count
		if tbl.RowCount != 5 {
			t.Errorf("expected row count 5, got %d", tbl.RowCount)
		}
	})
}

func TestNewPostgresDefaultsToPublicSchema(t *testing.T) {
	cfg := &config.SourceConfig{Type: "postgresql", Schema: ""}
	d, err := discovery.NewPostgres(cfg)
	if err != nil {
		t.Fatal(err)
	}
	// Can't directly check the private field, but confirm it doesn't error
	_ = d
}

func TestDiscoverWithoutConnectFails(t *testing.T) {
	cfg := &config.SourceConfig{Type: "postgresql", Host: "localhost", Port: 5432}
	d, err := discovery.NewPostgres(cfg)
	if err != nil {
		t.Fatal(err)
	}

	_, err = d.Discover(context.Background())
	if err == nil {
		t.Error("expected error when discovering without connecting")
	}
}
