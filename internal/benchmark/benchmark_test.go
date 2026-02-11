package benchmark

import (
	"context"
	"errors"
	"testing"
	"time"
)

// mockReader is a SourceReader that returns canned responses.
type mockReader struct {
	bytesRead int64
	elapsed   time.Duration
	err       error
}

func (m *mockReader) ReadSample(_ context.Context, _, _ string, _ float64) (int64, time.Duration, error) {
	return m.bytesRead, m.elapsed, m.err
}

func TestRun_ThroughputCalculation(t *testing.T) {
	reader := &mockReader{
		bytesRead: 100 * 1024 * 1024, // 100 MB
		elapsed:   10 * time.Second,   // 10 seconds
	}

	input := BenchmarkInput{
		TableName:      "orders",
		PartitionCol:   "id",
		TotalDataBytes: 10 * 1024 * 1024 * 1024, // 10 GB total
		MaxConnections: 20,
		SamplePercent:  1.0,
	}

	result, err := Run(context.Background(), reader, input)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	// 100 MB / 10s = 10 MB/s
	if result.ThroughputMBps < 9.9 || result.ThroughputMBps > 10.1 {
		t.Errorf("expected ~10 MB/s, got %.2f", result.ThroughputMBps)
	}

	if result.BytesRead != 100*1024*1024 {
		t.Errorf("expected 100 MB bytes read, got %d", result.BytesRead)
	}
}

func TestRun_Extrapolation(t *testing.T) {
	reader := &mockReader{
		bytesRead: 50 * 1024 * 1024, // 50 MB
		elapsed:   5 * time.Second,
	}

	input := BenchmarkInput{
		TableName:      "orders",
		TotalDataBytes: 100 * 1024 * 1024 * 1024, // 100 GB total
		SamplePercent:  1.0,
	}

	result, err := Run(context.Background(), reader, input)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	// Throughput: 50 MB / 5s = 10 MB/s
	// 100 GB at 10 MB/s = 10240 seconds ≈ 2h 50m
	expectedSeconds := float64(100*1024) / 10.0
	actualSeconds := result.EstimatedFullReadTime.Seconds()
	if actualSeconds < expectedSeconds*0.9 || actualSeconds > expectedSeconds*1.1 {
		t.Errorf("expected ~%.0fs, got %.0fs", expectedSeconds, actualSeconds)
	}
}

func TestRun_OneHourAchievable(t *testing.T) {
	tests := []struct {
		name       string
		totalBytes int64
		mbps       float64
		achievable bool
	}{
		{"small fast", 10 * 1024 * 1024 * 1024, 100, true},   // 10 GB at 100 MB/s ≈ 1.7m
		{"large slow", 500 * 1024 * 1024 * 1024, 10, false},   // 500 GB at 10 MB/s ≈ 14h
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bytesRead := int64(tt.mbps * 1024 * 1024) // 1 second of reading
			reader := &mockReader{
				bytesRead: bytesRead,
				elapsed:   time.Second,
			}

			input := BenchmarkInput{
				TableName:      "test",
				TotalDataBytes: tt.totalBytes,
				SamplePercent:  1.0,
			}

			result, err := Run(context.Background(), reader, input)
			if err != nil {
				t.Fatalf("Run: %v", err)
			}

			if result.OneHourAchievable != tt.achievable {
				t.Errorf("OneHourAchievable = %v, want %v (estimated: %v)",
					result.OneHourAchievable, tt.achievable, result.EstimatedFullReadTime)
			}
		})
	}
}

func TestRun_ContextCancellation(t *testing.T) {
	reader := &mockReader{
		err: context.Canceled,
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	input := BenchmarkInput{
		TableName:      "test",
		TotalDataBytes: 1024,
		SamplePercent:  1.0,
	}

	_, err := Run(ctx, reader, input)
	if err == nil {
		t.Error("expected error for cancelled context")
	}
}

func TestRun_ReaderError(t *testing.T) {
	reader := &mockReader{
		err: errors.New("connection refused"),
	}

	input := BenchmarkInput{
		TableName:      "test",
		TotalDataBytes: 1024,
		SamplePercent:  1.0,
	}

	_, err := Run(context.Background(), reader, input)
	if err == nil {
		t.Error("expected error from reader")
	}
}

func TestRun_DefaultSamplePercent(t *testing.T) {
	reader := &mockReader{
		bytesRead: 1024,
		elapsed:   time.Second,
	}

	input := BenchmarkInput{
		TableName:      "test",
		TotalDataBytes: 1024 * 1024,
	}

	result, err := Run(context.Background(), reader, input)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if result.Explanation == "" {
		t.Error("expected non-empty explanation")
	}
}
