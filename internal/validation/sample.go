package validation

import (
	"context"
	"fmt"

	"github.com/reloquent/reloquent/internal/mapping"
)

// SampleCheck holds the result of sample-based validation.
type SampleCheck struct {
	SampleSize    int              `json:"sample_size"`
	Checked       int              `json:"checked"`
	MismatchCount int              `json:"mismatch_count"`
	Mismatches    []SampleMismatch `json:"mismatches,omitempty"`
}

// SampleMismatch describes a field-level mismatch in a sampled document.
type SampleMismatch struct {
	DocumentID  interface{} `json:"document_id,omitempty"`
	Field       string      `json:"field"`
	SourceValue interface{} `json:"source_value"`
	TargetValue interface{} `json:"target_value"`
}

// validateSample samples documents from the target and checks that field values
// exist (basic presence check). Full reconstruction requires complex JOINs, so
// we validate that sampled documents have the expected top-level fields from the
// source table columns.
func (v *Validator) validateSample(ctx context.Context, col mapping.Collection) (*SampleCheck, error) {
	sampleSize := v.SampleSize
	if sampleSize <= 0 {
		sampleSize = 100
	}

	docs, err := v.Target.SampleDocuments(ctx, col.Name, sampleSize)
	if err != nil {
		return nil, fmt.Errorf("sampling documents from %s: %w", col.Name, err)
	}

	check := &SampleCheck{
		SampleSize: sampleSize,
		Checked:    len(docs),
	}

	// Get expected columns from the source table in the schema
	expectedFields := v.getExpectedFields(col)

	for _, doc := range docs {
		docID := doc["_id"]
		for _, field := range expectedFields {
			if _, ok := doc[field]; !ok {
				check.MismatchCount++
				check.Mismatches = append(check.Mismatches, SampleMismatch{
					DocumentID:  docID,
					Field:       field,
					SourceValue: "(expected)",
					TargetValue: "(missing)",
				})
			}
		}
	}

	return check, nil
}

// getExpectedFields returns the top-level fields expected in the target documents
// based on the source table columns.
func (v *Validator) getExpectedFields(col mapping.Collection) []string {
	if v.Schema == nil {
		return nil
	}
	for _, t := range v.Schema.Tables {
		if t.Name == col.SourceTable {
			fields := make([]string, 0, len(t.Columns))
			for _, c := range t.Columns {
				// Skip columns that are excluded via transformations
				excluded := false
				for _, tr := range col.Transformations {
					if tr.SourceField == c.Name && tr.Operation == "exclude" {
						excluded = true
						break
					}
				}
				if !excluded {
					fields = append(fields, c.Name)
				}
			}
			return fields
		}
	}
	return nil
}
