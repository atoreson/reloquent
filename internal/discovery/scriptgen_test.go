package discovery

import (
	"strings"
	"testing"
)

func TestPostgresScript_QueriesExpectedCatalogs(t *testing.T) {
	sg := &ScriptGenerator{DBType: "postgresql", Schema: "public"}
	script := sg.GenerateScript()

	catalogs := []string{
		"pg_class",
		"pg_namespace",
		"information_schema.columns",
		"information_schema.table_constraints",
		"information_schema.key_column_usage",
		"information_schema.constraint_column_usage",
	}
	for _, c := range catalogs {
		if !strings.Contains(script, c) {
			t.Errorf("PostgreSQL script should reference %s", c)
		}
	}
}

func TestOracleScript_QueriesExpectedCatalogs(t *testing.T) {
	sg := &ScriptGenerator{DBType: "oracle", Schema: "HR"}
	script := sg.GenerateScript()

	catalogs := []string{
		"ALL_TABLES",
		"ALL_TAB_COLUMNS",
		"ALL_CONSTRAINTS",
		"ALL_CONS_COLUMNS",
	}
	for _, c := range catalogs {
		if !strings.Contains(script, c) {
			t.Errorf("Oracle script should reference %s", c)
		}
	}
}

func TestPostgresScript_UsesSchemaName(t *testing.T) {
	sg := &ScriptGenerator{DBType: "postgresql", Schema: "myschema"}
	script := sg.GenerateScript()

	if !strings.Contains(script, "myschema") {
		t.Error("PostgreSQL script should use the specified schema name")
	}
}

func TestOracleScript_UsesOwner(t *testing.T) {
	sg := &ScriptGenerator{DBType: "oracle", Schema: "ADMIN"}
	script := sg.GenerateScript()

	if !strings.Contains(script, "ADMIN") {
		t.Error("Oracle script should use the specified owner")
	}
}

func TestPostgresWrapper_Structure(t *testing.T) {
	sg := &ScriptGenerator{DBType: "postgresql"}
	wrapper := sg.GenerateShellWrapper()

	if !strings.HasPrefix(wrapper, "#!/bin/bash") {
		t.Error("wrapper should start with shebang")
	}
	if !strings.Contains(wrapper, "psql") {
		t.Error("PostgreSQL wrapper should call psql")
	}
	if !strings.Contains(wrapper, "reloquent ingest") {
		t.Error("wrapper should mention ingest command")
	}
}

func TestOracleWrapper_Structure(t *testing.T) {
	sg := &ScriptGenerator{DBType: "oracle"}
	wrapper := sg.GenerateShellWrapper()

	if !strings.HasPrefix(wrapper, "#!/bin/bash") {
		t.Error("wrapper should start with shebang")
	}
	if !strings.Contains(wrapper, "sqlplus") {
		t.Error("Oracle wrapper should call sqlplus")
	}
}

func TestUnsupportedDBType(t *testing.T) {
	sg := &ScriptGenerator{DBType: "mysql"}
	script := sg.GenerateScript()

	if !strings.Contains(script, "Unsupported") {
		t.Error("unsupported type should produce a comment")
	}
}
