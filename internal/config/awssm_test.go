package config

import (
	"testing"
)

func TestResolveValue_AWSSM_Integration(t *testing.T) {
	// Without valid AWS credentials, this should fail gracefully
	// We test via ResolveValue to confirm the wiring is correct
	_, err := ResolveValue("${AWS_SM:nonexistent-secret}")
	if err == nil {
		t.Error("expected error when AWS credentials are not configured")
	}
}

func TestResolveValue_AWSSM_Pattern(t *testing.T) {
	// Verify the pattern matches correctly
	val, err := ResolveValue("plain-text-value")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != "plain-text-value" {
		t.Errorf("plain values should pass through, got %q", val)
	}
}
