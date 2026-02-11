package sizing

import "fmt"

// EMR sizing brackets from PLAN.md:
// 1-5 TB:   10-20  × r5.4xlarge  ($50-200)
// 5-20 TB:  20-50  × r5.8xlarge  ($100-500)
// 20-50 TB: 50-100 × r5.8xlarge  ($200-1000)
// 50-100 TB: 100-200 × r5.12xlarge ($500-3000)

type emrBracket struct {
	minBytes     int64
	maxBytes     int64
	instanceType string
	minWorkers   int
	maxWorkers   int
	costLow      float64
	costHigh     float64
}

var emrBrackets = []emrBracket{
	{minBytes: 0, maxBytes: tbToBytes(1), instanceType: "r5.4xlarge", minWorkers: 5, maxWorkers: 10, costLow: 25, costHigh: 100},
	{minBytes: tbToBytes(1), maxBytes: tbToBytes(5), instanceType: "r5.4xlarge", minWorkers: 10, maxWorkers: 20, costLow: 50, costHigh: 200},
	{minBytes: tbToBytes(5), maxBytes: tbToBytes(20), instanceType: "r5.8xlarge", minWorkers: 20, maxWorkers: 50, costLow: 100, costHigh: 500},
	{minBytes: tbToBytes(20), maxBytes: tbToBytes(50), instanceType: "r5.8xlarge", minWorkers: 50, maxWorkers: 100, costLow: 200, costHigh: 1000},
	{minBytes: tbToBytes(50), maxBytes: tbToBytes(100), instanceType: "r5.12xlarge", minWorkers: 100, maxWorkers: 200, costLow: 500, costHigh: 3000},
}

func calculateEMR(estimatedBytes int64) SparkPlan {
	bracket := emrBrackets[len(emrBrackets)-1] // default to largest
	for _, b := range emrBrackets {
		if estimatedBytes < b.maxBytes {
			bracket = b
			break
		}
	}

	// Interpolate worker count within bracket
	ratio := float64(estimatedBytes-bracket.minBytes) / float64(bracket.maxBytes-bracket.minBytes)
	if ratio < 0 {
		ratio = 0
	}
	if ratio > 1 {
		ratio = 1
	}
	workers := bracket.minWorkers + int(ratio*float64(bracket.maxWorkers-bracket.minWorkers))
	workers = clamp(workers, bracket.minWorkers, bracket.maxWorkers)

	costLow := bracket.costLow + ratio*(bracket.costHigh-bracket.costLow)*0.5
	costHigh := bracket.costLow + ratio*(bracket.costHigh-bracket.costLow)*1.2
	if costHigh > bracket.costHigh {
		costHigh = bracket.costHigh
	}

	return SparkPlan{
		Platform:     "emr",
		InstanceType: bracket.instanceType,
		WorkerCount:  workers,
		CostEstimate: fmt.Sprintf("$%.0f-$%.0f", costLow, costHigh),
		CostLow:      costLow,
		CostHigh:     costHigh,
	}
}
