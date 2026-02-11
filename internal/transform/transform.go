package transform

import (
	"fmt"
	"sort"
	"strings"

	"github.com/reloquent/reloquent/internal/mapping"
)

// Operation types for transformations.
const (
	OpRename  = "rename"
	OpCompute = "compute"
	OpCast    = "cast"
	OpFilter  = "filter"
	OpDefault = "default"
	OpExclude = "exclude"
)

// validOps is the set of valid operation names.
var validOps = map[string]bool{
	OpRename:  true,
	OpCompute: true,
	OpCast:    true,
	OpFilter:  true,
	OpDefault: true,
	OpExclude: true,
}

// operationOrder defines the execution ordering for transformations.
// Filters first (reduce data), then computes, renames, casts, defaults, excludes last.
var operationOrder = map[string]int{
	OpFilter:  0,
	OpCompute: 1,
	OpRename:  2,
	OpCast:    3,
	OpDefault: 4,
	OpExclude: 5,
}

// Validate checks that a single transformation is valid.
func Validate(t mapping.Transformation) error {
	if !validOps[t.Operation] {
		return fmt.Errorf("unknown operation %q", t.Operation)
	}

	switch t.Operation {
	case OpRename:
		if t.SourceField == "" {
			return fmt.Errorf("rename: source_field is required")
		}
		if t.TargetField == "" {
			return fmt.Errorf("rename: target_field is required")
		}
	case OpCompute:
		if t.TargetField == "" {
			return fmt.Errorf("compute: target_field is required")
		}
		if t.Expression == "" {
			return fmt.Errorf("compute: expression is required")
		}
	case OpCast:
		if t.SourceField == "" {
			return fmt.Errorf("cast: source_field is required")
		}
		if t.TargetType == "" {
			return fmt.Errorf("cast: target_type is required")
		}
	case OpFilter:
		if t.Expression == "" {
			return fmt.Errorf("filter: expression is required")
		}
	case OpDefault:
		if t.SourceField == "" {
			return fmt.Errorf("default: source_field is required")
		}
		if t.Value == "" {
			return fmt.Errorf("default: value is required")
		}
	case OpExclude:
		if t.SourceField == "" {
			return fmt.Errorf("exclude: source_field is required")
		}
	}

	return nil
}

// ValidateAll validates a slice of transformations and checks for conflicts.
func ValidateAll(transforms []mapping.Transformation) error {
	renamed := make(map[string]bool)
	excluded := make(map[string]bool)

	for i, t := range transforms {
		if err := Validate(t); err != nil {
			return fmt.Errorf("transformation %d: %w", i, err)
		}

		// Check conflicts
		switch t.Operation {
		case OpRename:
			if excluded[t.SourceField] {
				return fmt.Errorf("transformation %d: cannot rename excluded field %q", i, t.SourceField)
			}
			renamed[t.SourceField] = true
		case OpExclude:
			if renamed[t.SourceField] {
				return fmt.Errorf("transformation %d: cannot exclude renamed field %q", i, t.SourceField)
			}
			excluded[t.SourceField] = true
		}
	}

	return nil
}

// ToPySpark generates a PySpark code snippet for a single transformation.
func ToPySpark(t mapping.Transformation, dfName string) string {
	switch t.Operation {
	case OpRename:
		return fmt.Sprintf(`%s = %s.withColumnRenamed("%s", "%s")`,
			dfName, dfName, t.SourceField, t.TargetField)
	case OpCompute:
		return fmt.Sprintf(`%s = %s.withColumn("%s", expr("%s"))`,
			dfName, dfName, t.TargetField, t.Expression)
	case OpCast:
		return fmt.Sprintf(`%s = %s.withColumn("%s", col("%s").cast("%s"))`,
			dfName, dfName, t.SourceField, t.SourceField, t.TargetType)
	case OpFilter:
		return fmt.Sprintf(`%s = %s.filter("%s")`,
			dfName, dfName, t.Expression)
	case OpDefault:
		return fmt.Sprintf(`%s = %s.withColumn("%s", coalesce(col("%s"), lit(%s)))`,
			dfName, dfName, t.SourceField, t.SourceField, formatLiteral(t.Value))
	case OpExclude:
		return fmt.Sprintf(`%s = %s.drop("%s")`,
			dfName, dfName, t.SourceField)
	default:
		return fmt.Sprintf("# unknown operation: %s", t.Operation)
	}
}

// ToPySparkAll generates ordered PySpark code snippets for all transformations.
// Transformations are sorted by operation order: filter, compute, rename, cast, default, exclude.
func ToPySparkAll(transforms []mapping.Transformation, dfName string) []string {
	// Sort by operation order
	sorted := make([]mapping.Transformation, len(transforms))
	copy(sorted, transforms)
	sort.SliceStable(sorted, func(i, j int) bool {
		return operationOrder[sorted[i].Operation] < operationOrder[sorted[j].Operation]
	})

	lines := make([]string, 0, len(sorted))
	for _, t := range sorted {
		lines = append(lines, ToPySpark(t, dfName))
	}
	return lines
}

// formatLiteral formats a value as a Python literal for use in PySpark.
func formatLiteral(value string) string {
	// If it looks like a number, use as-is
	if isNumber(value) {
		return value
	}
	// Otherwise wrap in quotes
	return fmt.Sprintf(`"%s"`, strings.ReplaceAll(value, `"`, `\"`))
}

func isNumber(s string) bool {
	if s == "" {
		return false
	}
	dotSeen := false
	for i, c := range s {
		if c == '-' && i == 0 {
			continue
		}
		if c == '.' && !dotSeen {
			dotSeen = true
			continue
		}
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}
