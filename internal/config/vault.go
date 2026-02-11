package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/hashicorp/vault/api"
)

// resolveVault resolves a Vault secret reference.
// Format: secret/data/path#key
func resolveVault(ref string) (string, error) {
	parts := strings.SplitN(ref, "#", 2)
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid Vault reference %q: expected format path#key", ref)
	}
	path := parts[0]
	key := parts[1]

	addr := os.Getenv("VAULT_ADDR")
	if addr == "" {
		return "", fmt.Errorf("VAULT_ADDR environment variable not set")
	}

	token := os.Getenv("VAULT_TOKEN")
	if token == "" {
		return "", fmt.Errorf("VAULT_TOKEN environment variable not set")
	}

	cfg := api.DefaultConfig()
	cfg.Address = addr

	client, err := api.NewClient(cfg)
	if err != nil {
		return "", fmt.Errorf("creating Vault client: %w", err)
	}
	client.SetToken(token)

	secret, err := client.Logical().Read(path)
	if err != nil {
		return "", fmt.Errorf("reading Vault secret at %s: %w", path, err)
	}
	if secret == nil || secret.Data == nil {
		return "", fmt.Errorf("no secret found at %s", path)
	}

	// KV v2 stores data under a "data" sub-key
	data := secret.Data
	if innerData, ok := data["data"].(map[string]interface{}); ok {
		data = innerData
	}

	val, ok := data[key]
	if !ok {
		return "", fmt.Errorf("key %q not found in Vault secret at %s", key, path)
	}

	str, ok := val.(string)
	if !ok {
		return "", fmt.Errorf("Vault secret value for key %q is not a string", key)
	}

	return str, nil
}
