package sizing

import (
	"fmt"
	"time"
)

// Explanation provides a plain-language explanation of a sizing recommendation.
type Explanation struct {
	Category string `yaml:"category"` // "spark", "mongodb", "time", "overview"
	Summary  string `yaml:"summary"`
	Detail   string `yaml:"detail"`
}

func generateExplanations(input Input, spark SparkPlan, mongo MongoPlan, estTime time.Duration) []Explanation {
	estimatedBytes := int64(float64(input.TotalDataBytes) * input.DenormExpansionFactor)

	var explanations []Explanation

	// Overview
	explanations = append(explanations, Explanation{
		Category: "overview",
		Summary:  fmt.Sprintf("Migrating %s of data (%d rows across %d collections)", FormatBytes(estimatedBytes), input.TotalRowCount, input.CollectionCount),
		Detail: fmt.Sprintf(
			"Your source database contains %s of raw data. After denormalization (embedding child documents into parents), "+
				"the estimated target size is %s (%.1fx expansion factor). This is like packing boxes for a move — "+
				"when you combine items from separate drawers into single boxes, you need a bit more total space due to packaging overhead.",
			FormatBytes(input.TotalDataBytes), FormatBytes(estimatedBytes), input.DenormExpansionFactor),
	})

	// Spark cluster
	if spark.Platform == "glue" {
		explanations = append(explanations, Explanation{
			Category: "spark",
			Summary:  fmt.Sprintf("AWS Glue with %d DPUs (%s)", spark.DPUCount, spark.CostEstimate),
			Detail: fmt.Sprintf(
				"For %s of data, AWS Glue is recommended as it's simpler to operate than EMR — no cluster management needed. "+
					"With %d DPUs (Data Processing Units), Glue can process your data efficiently. Think of DPUs like workers on an assembly line: "+
					"each one reads from your source database and writes to MongoDB in parallel. Estimated cost: %s.",
				FormatBytes(estimatedBytes), spark.DPUCount, spark.CostEstimate),
		})
	} else {
		explanations = append(explanations, Explanation{
			Category: "spark",
			Summary:  fmt.Sprintf("EMR cluster: %d × %s (%s)", spark.WorkerCount, spark.InstanceType, spark.CostEstimate),
			Detail: fmt.Sprintf(
				"For %s of data, an EMR cluster with %d worker nodes of type %s is recommended. "+
					"Each worker reads from your source database in parallel and writes to MongoDB. Think of it like a team of movers: "+
					"more workers means the job finishes faster, but each worker costs money per hour. The cluster is transient — "+
					"it runs only during migration and is terminated afterward. Estimated cost: %s.",
				FormatBytes(estimatedBytes), spark.WorkerCount, spark.InstanceType, spark.CostEstimate),
		})
	}

	// MongoDB tier
	explanations = append(explanations, Explanation{
		Category: "mongodb",
		Summary:  fmt.Sprintf("Migration: %s → Production: %s (%d GB storage)", mongo.MigrationTier, mongo.ProductionTier, mongo.StorageGB),
		Detail: fmt.Sprintf(
			"During migration, we recommend a larger tier (%s) to handle the high write throughput — "+
				"like widening a highway during rush hour. After migration completes, you can scale down to %s "+
				"for normal operations. Estimated storage needed: %d GB (includes indexes, padding, and overhead at 1.5x raw data size).",
			mongo.MigrationTier, mongo.ProductionTier, mongo.StorageGB),
	})

	// Time estimate
	timeDesc := "without a benchmark"
	if input.BenchmarkMBps > 0 {
		timeDesc = fmt.Sprintf("based on measured %.0f MB/s throughput", input.BenchmarkMBps)
	}
	explanations = append(explanations, Explanation{
		Category: "time",
		Summary:  fmt.Sprintf("Estimated migration time: %s (%s)", FormatDuration(estTime), timeDesc),
		Detail: fmt.Sprintf(
			"The estimated migration duration is %s. This estimate is %s. "+
				"Actual time depends on network bandwidth between your source database, the Spark cluster, and MongoDB, "+
				"as well as the complexity of your denormalization rules. Running the benchmark (reloquent estimate --benchmark) "+
				"gives a more accurate estimate by measuring actual read throughput from your source database.",
			FormatDuration(estTime), timeDesc),
	})

	return explanations
}
