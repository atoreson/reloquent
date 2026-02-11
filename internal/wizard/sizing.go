package wizard

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/reloquent/reloquent/internal/benchmark"
	"github.com/reloquent/reloquent/internal/sizing"
)

// SizingModel is the bubbletea model for Step 6: Sizing.
type SizingModel struct {
	plan        *sizing.SizingPlan
	benchResult *benchmark.Result
	cursor      int
	done        bool
	cancelled   bool
	width       int
	height      int
}

// NewSizingModel creates a sizing model with a pre-computed plan.
func NewSizingModel(plan *sizing.SizingPlan) SizingModel {
	return SizingModel{
		plan:   plan,
		width:  100,
		height: 24,
	}
}

func (m SizingModel) Init() tea.Cmd {
	return nil
}

func (m SizingModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "enter", "f":
			m.done = true
			return m, tea.Quit
		case "q", "esc":
			m.done = true
			m.cancelled = true
			return m, tea.Quit
		case "b":
			// Toggle benchmark (placeholder — actual benchmark is run externally)
			return m, nil
		}
	}

	return m, nil
}

func (m SizingModel) View() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("Step 6: Sizing Recommendations"))
	b.WriteString("\n\n")

	if m.plan == nil {
		b.WriteString("  No sizing plan available.\n")
		b.WriteString(dimStyle.Render("\n  Press enter to continue, q to cancel"))
		return b.String()
	}

	// Display explanations
	for _, exp := range m.plan.Explanations {
		b.WriteString(fmt.Sprintf("  %s  %s\n", highlightStyle.Render(exp.Category), exp.Summary))
		b.WriteString(fmt.Sprintf("  %s\n\n", dimStyle.Render(exp.Detail)))
	}

	// Spark plan summary
	sp := m.plan.SparkPlan
	b.WriteString(highlightStyle.Render("  Spark Cluster:"))
	if sp.Platform == "glue" {
		b.WriteString(fmt.Sprintf(" AWS Glue, %d DPUs, %s\n", sp.DPUCount, sp.CostEstimate))
	} else {
		b.WriteString(fmt.Sprintf(" EMR, %d × %s, %s\n", sp.WorkerCount, sp.InstanceType, sp.CostEstimate))
	}

	// MongoDB plan summary
	mp := m.plan.MongoPlan
	b.WriteString(highlightStyle.Render("  MongoDB:"))
	b.WriteString(fmt.Sprintf(" Migration: %s → Production: %s (%d GB)\n", mp.MigrationTier, mp.ProductionTier, mp.StorageGB))

	// Estimated time
	b.WriteString(highlightStyle.Render("  Duration:"))
	b.WriteString(fmt.Sprintf(" %s\n", sizing.FormatDuration(m.plan.EstimatedTime)))

	// Sharding
	if m.plan.ShardPlan != nil && m.plan.ShardPlan.Recommended {
		b.WriteString(highlightStyle.Render("  Sharding:"))
		b.WriteString(fmt.Sprintf(" %d shards recommended\n", m.plan.ShardPlan.ShardCount))
	}

	// Benchmark result
	if m.benchResult != nil {
		b.WriteString("\n")
		b.WriteString(successStyle.Render("  Benchmark:"))
		b.WriteString(fmt.Sprintf(" %.1f MB/s from %s\n", m.benchResult.ThroughputMBps, m.benchResult.TableName))
	}

	b.WriteString("\n")
	b.WriteString(dimStyle.Render("  b: run benchmark  enter: continue  q: cancel"))

	return b.String()
}

// Done returns true when the model is finished.
func (m SizingModel) Done() bool {
	return m.done
}

// Cancelled returns true if the user cancelled.
func (m SizingModel) Cancelled() bool {
	return m.cancelled
}

// SetBenchmarkResult sets the benchmark result for display.
func (m *SizingModel) SetBenchmarkResult(result *benchmark.Result) {
	m.benchResult = result
}
