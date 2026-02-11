//go:build integration

package integration

import (
	"fmt"
	"os"
	"testing"
)

func pgConnString(t *testing.T) string {
	t.Helper()
	host := envOrDefault("RELOQUENT_TEST_PG_HOST", "localhost")
	port := envOrDefault("RELOQUENT_TEST_PG_PORT", "25432")
	db := envOrDefault("RELOQUENT_TEST_PG_DATABASE", "reloquent_test")
	user := envOrDefault("RELOQUENT_TEST_PG_USER", "postgres")
	pass := envOrDefault("RELOQUENT_TEST_PG_PASSWORD", "postgres")
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", user, pass, host, port, db)
}

func pgHost(t *testing.T) string {
	t.Helper()
	return envOrDefault("RELOQUENT_TEST_PG_HOST", "localhost")
}

func pgPort(t *testing.T) int {
	t.Helper()
	p := envOrDefault("RELOQUENT_TEST_PG_PORT", "25432")
	var port int
	fmt.Sscanf(p, "%d", &port)
	return port
}

func pgDatabase(t *testing.T) string {
	t.Helper()
	return envOrDefault("RELOQUENT_TEST_PG_DATABASE", "reloquent_test")
}

func pgUser(t *testing.T) string {
	t.Helper()
	return envOrDefault("RELOQUENT_TEST_PG_USER", "postgres")
}

func pgPassword(t *testing.T) string {
	t.Helper()
	return envOrDefault("RELOQUENT_TEST_PG_PASSWORD", "postgres")
}

func mongoURI(t *testing.T) string {
	t.Helper()
	return envOrDefault("RELOQUENT_TEST_MONGO_URI", "mongodb://localhost:37017/?directConnection=true")
}

func mongoDatabase(t *testing.T) string {
	t.Helper()
	return envOrDefault("RELOQUENT_TEST_MONGO_DATABASE", "reloquent_test")
}

func skipIfNoPostgres(t *testing.T) {
	t.Helper()
	if os.Getenv("RELOQUENT_TEST_PG_HOST") == "" && os.Getenv("RELOQUENT_TEST_PG_PORT") == "" {
		t.Skip("skipping: RELOQUENT_TEST_PG_HOST/PORT not set")
	}
}

func skipIfNoMongo(t *testing.T) {
	t.Helper()
	if os.Getenv("RELOQUENT_TEST_MONGO_URI") == "" {
		t.Skip("skipping: RELOQUENT_TEST_MONGO_URI not set")
	}
}

func envOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
