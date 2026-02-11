package wizard

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/reloquent/reloquent/internal/sizing"
)

// ReviewModel is the bubbletea model for Step 8b: Review & Confirm.
type ReviewModel struct {
	plan       *sizing.SizingPlan
	script     string
	showScript bool
	confirmed  bool
	done       bool
	cancelled  bool
	width      int
	height     int
}

// NewReviewModel creates a review model.
func NewReviewModel(plan *sizing.SizingPlan, script string) ReviewModel {
	return ReviewModel{
		plan:   plan,
		script: script,
		width:  100,
		height: 24,
	}
}

func (m ReviewModel) Init() tea.Cmd {
	return nil
}

func (m ReviewModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			m.done = true
			m.confirmed = true
			return m, tea.Quit
		case "q", "esc":
			m.done = true
			m.cancelled = true
			return m, tea.Quit
		case "v":
			m.showScript = !m.showScript
			return m, nil
		}
	}

	return m, nil
}

func (m ReviewModel) View() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("Step 8b: Review & Confirm"))
	b.WriteString("\n\n")

	// Migration summary
	b.WriteString(highlightStyle.Render("  Migration Summary"))
	b.WriteString("\n\n")

	if m.plan != nil {
		sp := m.plan.SparkPlan
		if sp.Platform == "glue" {
			b.WriteString(fmt.Sprintf("  Platform:  AWS Glue (%d DPUs)\n", sp.DPUCount))
		} else {
			b.WriteString(fmt.Sprintf("  Platform:  EMR (%d × %s)\n", sp.WorkerCount, sp.InstanceType))
		}
		b.WriteString(fmt.Sprintf("  Cost:      %s\n", sp.CostEstimate))

		mp := m.plan.MongoPlan
		b.WriteString(fmt.Sprintf("  Target:    %s → %s\n", mp.MigrationTier, mp.ProductionTier))
		b.WriteString(fmt.Sprintf("  Storage:   %d GB\n", mp.StorageGB))
		b.WriteString(fmt.Sprintf("  Duration:  %s\n", sizing.FormatDuration(m.plan.EstimatedTime)))

		if m.plan.ShardPlan != nil && m.plan.ShardPlan.Recommended {
			b.WriteString(fmt.Sprintf("  Sharding:  %d shards\n", m.plan.ShardPlan.ShardCount))
		}
	}

	// Script toggle
	b.WriteString("\n")
	if m.showScript {
		b.WriteString(highlightStyle.Render("  PySpark Script:"))
		b.WriteString("\n")
		// Show script with line numbers, limited to visible height
		lines := strings.Split(m.script, "\n")
		maxLines := m.height - 20
		if maxLines < 10 {
			maxLines = 10
		}
		for i, line := range lines {
			if i >= maxLines {
				b.WriteString(dimStyle.Render(fmt.Sprintf("  ... (%d more lines)\n", len(lines)-maxLines)))
				break
			}
			b.WriteString(fmt.Sprintf("  %3d  %s\n", i+1, line))
		}
	} else {
		b.WriteString(dimStyle.Render("  Press v to view PySpark script"))
		b.WriteString("\n")
	}

	// Point-of-no-return warning
	b.WriteString("\n")
	b.WriteString(errStyle.Render("  WARNING: Pressing enter will start the migration."))
	b.WriteString("\n")
	b.WriteString(errStyle.Render("  This will write data to the target MongoDB cluster."))
	b.WriteString("\n\n")

	b.WriteString(dimStyle.Render("  v: toggle script  enter: START MIGRATION  q: go back"))

	return b.String()
}

// Done returns true when the model is finished.
func (m ReviewModel) Done() bool {
	return m.done
}

// Cancelled returns true if the user cancelled.
func (m ReviewModel) Cancelled() bool {
	return m.cancelled
}

// Confirmed returns true if the user confirmed the migration.
func (m ReviewModel) Confirmed() bool {
	return m.confirmed
}
