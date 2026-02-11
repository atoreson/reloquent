package sizing

import "fmt"

// Glue sizing brackets:
// <50 GB:    10 DPU
// 50-250 GB: 50-100 DPU
// 250-500 GB: 150-299 DPU
// >500 GB:  not recommended (use EMR)

const (
	glueMaxRecommendedBytes = 500 * 1024 * 1024 * 1024 // 500 GB
	gluePricePerDPUHour     = 0.44                      // USD
)

type glueBracket struct {
	minBytes int64
	maxBytes int64
	minDPU   int
	maxDPU   int
}

var glueBrackets = []glueBracket{
	{minBytes: 0, maxBytes: gbToBytes(50), minDPU: 10, maxDPU: 10},
	{minBytes: gbToBytes(50), maxBytes: gbToBytes(250), minDPU: 50, maxDPU: 100},
	{minBytes: gbToBytes(250), maxBytes: gbToBytes(500), minDPU: 150, maxDPU: 299},
}

// IsGlueViable returns true if the data size is within Glue's recommended range.
func IsGlueViable(estimatedBytes int64) bool {
	return estimatedBytes <= glueMaxRecommendedBytes
}

func calculateGlue(estimatedBytes int64) SparkPlan {
	if !IsGlueViable(estimatedBytes) {
		return SparkPlan{} // not viable, DPUCount=0 signals this
	}

	bracket := glueBrackets[0]
	for _, b := range glueBrackets {
		if estimatedBytes >= b.minBytes && estimatedBytes < b.maxBytes {
			bracket = b
			break
		}
		bracket = b
	}

	// Interpolate DPU count within bracket
	ratio := float64(estimatedBytes-bracket.minBytes) / float64(bracket.maxBytes-bracket.minBytes)
	if ratio < 0 {
		ratio = 0
	}
	if ratio > 1 {
		ratio = 1
	}
	dpus := bracket.minDPU + int(ratio*float64(bracket.maxDPU-bracket.minDPU))
	dpus = clamp(dpus, bracket.minDPU, bracket.maxDPU)

	// Estimate cost: assume 1-3 hours for migration
	costLow := float64(dpus) * gluePricePerDPUHour * 1
	costHigh := float64(dpus) * gluePricePerDPUHour * 3

	return SparkPlan{
		Platform:     "glue",
		DPUCount:     dpus,
		CostEstimate: fmt.Sprintf("$%.0f-$%.0f", costLow, costHigh),
		CostLow:      costLow,
		CostHigh:     costHigh,
	}
}
