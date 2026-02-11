package validation

import (
	"context"
	"fmt"
	"math"

	"github.com/reloquent/reloquent/internal/mapping"
)

// AggregateCheck holds the result of aggregate comparison.
type AggregateCheck struct {
	Match  bool              `json:"match"`
	Checks []AggregateDetail `json:"checks,omitempty"`
}

// AggregateDetail describes a single aggregate comparison.
type AggregateDetail struct {
	Type        string  `json:"type"` // "count_distinct" or "sum"
	Column      string  `json:"column"`
	SourceValue float64 `json:"source_value"`
	TargetValue float64 `json:"target_value"`
	Match       bool    `json:"match"`
}

// validateAggregates runs aggregate comparisons for the primary key column.
// COUNT(DISTINCT pk) on source should equal COUNT(DISTINCT pk) on target.
func (v *Validator) validateAggregates(ctx context.Context, col mapping.Collection) (*AggregateCheck, error) {
	check := &AggregateCheck{Match: true}

	// Find the primary key column for this source table
	pkColumn := v.findPKColumn(col.SourceTable)
	if pkColumn == "" {
		// No PK found, skip aggregate check
		return check, nil
	}

	// COUNT DISTINCT on PK
	sourceDistinct, err := v.Source.AggregateCountDistinct(ctx, col.SourceTable, pkColumn)
	if err != nil {
		return nil, fmt.Errorf("source count distinct %s.%s: %w", col.SourceTable, pkColumn, err)
	}

	targetDistinct, err := v.Target.AggregateCountDistinct(ctx, col.Name, pkColumn)
	if err != nil {
		return nil, fmt.Errorf("target count distinct %s.%s: %w", col.Name, pkColumn, err)
	}

	cdMatch := sourceDistinct == targetDistinct
	check.Checks = append(check.Checks, AggregateDetail{
		Type:        "count_distinct",
		Column:      pkColumn,
		SourceValue: float64(sourceDistinct),
		TargetValue: float64(targetDistinct),
		Match:       cdMatch,
	})
	if !cdMatch {
		check.Match = false
	}

	// Find numeric columns for SUM comparison
	numericCols := v.findNumericColumns(col.SourceTable)
	for _, nc := range numericCols {
		sourceSum, err := v.Source.AggregateSum(ctx, col.SourceTable, nc)
		if err != nil {
			return nil, fmt.Errorf("source sum %s.%s: %w", col.SourceTable, nc, err)
		}

		targetSum, err := v.Target.AggregateSum(ctx, col.Name, nc)
		if err != nil {
			return nil, fmt.Errorf("target sum %s.%s: %w", col.Name, nc, err)
		}

		sumMatch := floatClose(sourceSum, targetSum)
		check.Checks = append(check.Checks, AggregateDetail{
			Type:        "sum",
			Column:      nc,
			SourceValue: sourceSum,
			TargetValue: targetSum,
			Match:       sumMatch,
		})
		if !sumMatch {
			check.Match = false
		}
	}

	return check, nil
}

func (v *Validator) findPKColumn(tableName string) string {
	if v.Schema == nil {
		return ""
	}
	for _, t := range v.Schema.Tables {
		if t.Name == tableName && t.PrimaryKey != nil && len(t.PrimaryKey.Columns) > 0 {
			return t.PrimaryKey.Columns[0]
		}
	}
	return ""
}

func (v *Validator) findNumericColumns(tableName string) []string {
	if v.Schema == nil {
		return nil
	}
	numericTypes := map[string]bool{
		"integer": true, "int": true, "int4": true, "int8": true,
		"bigint": true, "smallint": true, "serial": true, "bigserial": true,
		"numeric": true, "decimal": true, "real": true, "float": true,
		"float4": true, "float8": true, "double precision": true,
		"number": true, "binary_float": true, "binary_double": true,
	}

	for _, t := range v.Schema.Tables {
		if t.Name != tableName {
			continue
		}
		var cols []string
		for _, c := range t.Columns {
			if numericTypes[c.DataType] {
				// Skip PK columns (already checked via count distinct)
				if t.PrimaryKey != nil {
					isPK := false
					for _, pc := range t.PrimaryKey.Columns {
						if pc == c.Name {
							isPK = true
							break
						}
					}
					if isPK {
						continue
					}
				}
				cols = append(cols, c.Name)
			}
		}
		return cols
	}
	return nil
}

// floatClose checks if two floats are approximately equal (within 0.01% relative tolerance).
func floatClose(a, b float64) bool {
	if a == b {
		return true
	}
	if a == 0 || b == 0 {
		return math.Abs(a-b) < 0.01
	}
	return math.Abs(a-b)/math.Max(math.Abs(a), math.Abs(b)) < 0.0001
}
