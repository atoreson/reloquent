package validation

import (
	"context"
	"fmt"

	"github.com/reloquent/reloquent/internal/mapping"
)

// RowCountCheck holds the result of a row count comparison.
type RowCountCheck struct {
	SourceCount int64  `json:"source_count"`
	TargetCount int64  `json:"target_count"`
	Match       bool   `json:"match"`
	Message     string `json:"message,omitempty"`
}

// validateRowCount compares the source table row count against the target collection document count.
// For denormalized collections: expected count = root table row count (embedded children don't add documents).
func (v *Validator) validateRowCount(ctx context.Context, col mapping.Collection) (*RowCountCheck, error) {
	sourceCount, err := v.Source.RowCount(ctx, col.SourceTable)
	if err != nil {
		return nil, fmt.Errorf("counting source rows for %s: %w", col.SourceTable, err)
	}

	targetCount, err := v.Target.CountDocuments(ctx, col.Name)
	if err != nil {
		return nil, fmt.Errorf("counting target docs for %s: %w", col.Name, err)
	}

	check := &RowCountCheck{
		SourceCount: sourceCount,
		TargetCount: targetCount,
		Match:       sourceCount == targetCount,
	}

	if !check.Match {
		check.Message = fmt.Sprintf("count mismatch: source=%d, target=%d (diff=%d)",
			sourceCount, targetCount, sourceCount-targetCount)
	}

	return check, nil
}
