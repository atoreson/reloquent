package validation

import (
	"context"
	"time"

	"github.com/reloquent/reloquent/internal/mapping"
	"github.com/reloquent/reloquent/internal/schema"
	"github.com/reloquent/reloquent/internal/source"
	"github.com/reloquent/reloquent/internal/target"
)

// Result holds the outcome of post-migration validation.
type Result struct {
	Status      string             `json:"status"` // PASS, FAIL, PARTIAL
	Collections []CollectionResult `json:"collections"`
	StartedAt   time.Time          `json:"started_at"`
	CompletedAt time.Time          `json:"completed_at"`
}

// CollectionResult holds validation results for a single collection.
type CollectionResult struct {
	Name           string          `json:"name"`
	RowCountCheck  *RowCountCheck  `json:"row_count_check,omitempty"`
	SampleCheck    *SampleCheck    `json:"sample_check,omitempty"`
	AggregateCheck *AggregateCheck `json:"aggregate_check,omitempty"`
	Status         string          `json:"status"` // PASS, FAIL
}

// Validator performs post-migration validation.
type Validator struct {
	Source     source.Reader
	Target     target.Operator
	Schema     *schema.Schema
	Mapping    *mapping.Mapping
	SampleSize int
	Callback   func(collection, checkType string, passed bool)
}

// Validate runs all validation checks: row counts, samples, and aggregates.
func (v *Validator) Validate(ctx context.Context) (*Result, error) {
	result := &Result{
		StartedAt: time.Now(),
	}

	for _, col := range v.Mapping.Collections {
		cr := CollectionResult{Name: col.Name, Status: "PASS"}

		// Row count check
		rc, err := v.validateRowCount(ctx, col)
		if err != nil {
			return nil, err
		}
		cr.RowCountCheck = rc
		if !rc.Match {
			cr.Status = "FAIL"
		}
		v.notify(col.Name, "row_count", rc.Match)

		// Sample check
		sc, err := v.validateSample(ctx, col)
		if err != nil {
			return nil, err
		}
		cr.SampleCheck = sc
		if sc.MismatchCount > 0 {
			cr.Status = "FAIL"
		}
		v.notify(col.Name, "sample", sc.MismatchCount == 0)

		// Aggregate check
		ac, err := v.validateAggregates(ctx, col)
		if err != nil {
			return nil, err
		}
		cr.AggregateCheck = ac
		if !ac.Match {
			cr.Status = "FAIL"
		}
		v.notify(col.Name, "aggregate", ac.Match)

		result.Collections = append(result.Collections, cr)
	}

	result.CompletedAt = time.Now()
	result.Status = computeOverallStatus(result.Collections)
	return result, nil
}

// ValidateRowCounts runs only the row count validation.
func (v *Validator) ValidateRowCounts(ctx context.Context) (*Result, error) {
	result := &Result{StartedAt: time.Now()}

	for _, col := range v.Mapping.Collections {
		cr := CollectionResult{Name: col.Name, Status: "PASS"}
		rc, err := v.validateRowCount(ctx, col)
		if err != nil {
			return nil, err
		}
		cr.RowCountCheck = rc
		if !rc.Match {
			cr.Status = "FAIL"
		}
		v.notify(col.Name, "row_count", rc.Match)
		result.Collections = append(result.Collections, cr)
	}

	result.CompletedAt = time.Now()
	result.Status = computeOverallStatus(result.Collections)
	return result, nil
}

// ValidateSamples runs only the sample validation.
func (v *Validator) ValidateSamples(ctx context.Context) (*Result, error) {
	result := &Result{StartedAt: time.Now()}

	for _, col := range v.Mapping.Collections {
		cr := CollectionResult{Name: col.Name, Status: "PASS"}
		sc, err := v.validateSample(ctx, col)
		if err != nil {
			return nil, err
		}
		cr.SampleCheck = sc
		if sc.MismatchCount > 0 {
			cr.Status = "FAIL"
		}
		v.notify(col.Name, "sample", sc.MismatchCount == 0)
		result.Collections = append(result.Collections, cr)
	}

	result.CompletedAt = time.Now()
	result.Status = computeOverallStatus(result.Collections)
	return result, nil
}

// ValidateAggregates runs only the aggregate validation.
func (v *Validator) ValidateAggregates(ctx context.Context) (*Result, error) {
	result := &Result{StartedAt: time.Now()}

	for _, col := range v.Mapping.Collections {
		cr := CollectionResult{Name: col.Name, Status: "PASS"}
		ac, err := v.validateAggregates(ctx, col)
		if err != nil {
			return nil, err
		}
		cr.AggregateCheck = ac
		if !ac.Match {
			cr.Status = "FAIL"
		}
		v.notify(col.Name, "aggregate", ac.Match)
		result.Collections = append(result.Collections, cr)
	}

	result.CompletedAt = time.Now()
	result.Status = computeOverallStatus(result.Collections)
	return result, nil
}

func (v *Validator) notify(collection, checkType string, passed bool) {
	if v.Callback != nil {
		v.Callback(collection, checkType, passed)
	}
}

func computeOverallStatus(collections []CollectionResult) string {
	if len(collections) == 0 {
		return "PASS"
	}
	failCount := 0
	for _, c := range collections {
		if c.Status == "FAIL" {
			failCount++
		}
	}
	if failCount == 0 {
		return "PASS"
	}
	if failCount == len(collections) {
		return "FAIL"
	}
	return "PARTIAL"
}
