package benchmark

import (
	"context"
	"fmt"
	"time"
)

// SourceReader reads sample data from a source database for benchmarking.
type SourceReader interface {
	ReadSample(ctx context.Context, tableName, partitionCol string, samplePct float64) (bytesRead int64, elapsed time.Duration, err error)
}

// Result holds the output of a benchmark run.
type Result struct {
	TableName             string        `yaml:"table_name"`
	BytesRead             int64         `yaml:"bytes_read"`
	Elapsed               time.Duration `yaml:"elapsed"`
	ThroughputMBps        float64       `yaml:"throughput_mbps"`
	Connections           int           `yaml:"connections"`
	EstimatedFullReadTime time.Duration `yaml:"estimated_full_read_time"`
	OneHourAchievable     bool          `yaml:"one_hour_achievable"`
	Explanation           string        `yaml:"explanation"`
}

// BenchmarkInput defines parameters for a benchmark run.
type BenchmarkInput struct {
	TableName      string
	PartitionCol   string
	TotalDataBytes int64
	MaxConnections int
	SamplePercent  float64 // default 1.0%
}

// Run executes a benchmark against the given source database.
func Run(ctx context.Context, reader SourceReader, input BenchmarkInput) (*Result, error) {
	if input.SamplePercent == 0 {
		input.SamplePercent = 1.0
	}
	if input.MaxConnections == 0 {
		input.MaxConnections = 20
	}

	bytesRead, elapsed, err := reader.ReadSample(ctx, input.TableName, input.PartitionCol, input.SamplePercent)
	if err != nil {
		return nil, fmt.Errorf("reading sample from %s: %w", input.TableName, err)
	}

	if elapsed == 0 {
		elapsed = time.Millisecond // avoid division by zero
	}

	// Calculate throughput from the sample
	throughputMBps := float64(bytesRead) / (1024 * 1024) / elapsed.Seconds()

	// Extrapolate: the sample was SamplePercent of the data, so scale up
	// but throughput stays the same (it's a rate)
	var estFullRead time.Duration
	if throughputMBps > 0 {
		totalMB := float64(input.TotalDataBytes) / (1024 * 1024)
		estFullRead = time.Duration(totalMB/throughputMBps) * time.Second
	}

	oneHour := estFullRead <= time.Hour

	explanation := fmt.Sprintf(
		"Read %s in %s from table '%s' (%.1f%% sample). "+
			"Measured throughput: %.1f MB/s. "+
			"Estimated full read time: %s.",
		formatBytes(bytesRead),
		formatDuration(elapsed),
		input.TableName,
		input.SamplePercent,
		throughputMBps,
		formatDuration(estFullRead),
	)

	if oneHour {
		explanation += " Full migration achievable within 1 hour."
	} else {
		explanation += fmt.Sprintf(" Full migration estimated at %s — consider increasing parallelism or migration window.", formatDuration(estFullRead))
	}

	return &Result{
		TableName:             input.TableName,
		BytesRead:             bytesRead,
		Elapsed:               elapsed,
		ThroughputMBps:        throughputMBps,
		Connections:           input.MaxConnections,
		EstimatedFullReadTime: estFullRead,
		OneHourAchievable:     oneHour,
		Explanation:           explanation,
	}, nil
}

func formatBytes(b int64) string {
	const (
		kb = 1024
		mb = kb * 1024
		gb = mb * 1024
	)
	switch {
	case b >= gb:
		return fmt.Sprintf("%.1f GB", float64(b)/float64(gb))
	case b >= mb:
		return fmt.Sprintf("%.1f MB", float64(b)/float64(mb))
	case b >= kb:
		return fmt.Sprintf("%.1f KB", float64(b)/float64(kb))
	default:
		return fmt.Sprintf("%d B", b)
	}
}

func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	if d < time.Hour {
		return fmt.Sprintf("%.0fm", d.Minutes())
	}
	hours := int(d.Hours())
	mins := int(d.Minutes()) % 60
	if mins == 0 {
		return fmt.Sprintf("%dh", hours)
	}
	return fmt.Sprintf("%dh %dm", hours, mins)
}
