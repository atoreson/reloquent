package wizard

import (
	"fmt"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestNewTargetModel(t *testing.T) {
	m := NewTargetModel()
	if m.focused != targetFieldConnStr {
		t.Errorf("expected focus on connection string field, got %d", m.focused)
	}
	if m.done {
		t.Error("should not be done initially")
	}
	if m.connecting {
		t.Error("should not be connecting initially")
	}
	if m.result != nil {
		t.Error("result should be nil initially")
	}
}

func TestTargetFieldNavigation(t *testing.T) {
	m := NewTargetModel()

	// Tab forward
	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = result.(TargetModel)
	if m.focused != targetFieldDatabase {
		t.Errorf("after tab: expected focused=%d, got %d", targetFieldDatabase, m.focused)
	}

	// Tab wraps around
	result, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = result.(TargetModel)
	if m.focused != targetFieldConnStr {
		t.Errorf("after second tab: expected focused=%d, got %d", targetFieldConnStr, m.focused)
	}

	// Shift-tab wraps backwards
	result, _ = m.Update(tea.KeyMsg{Type: tea.KeyShiftTab})
	m = result.(TargetModel)
	if m.focused != targetFieldDatabase {
		t.Errorf("after shift-tab: expected focused=%d, got %d", targetFieldDatabase, m.focused)
	}
}

func TestTargetCancel(t *testing.T) {
	m := NewTargetModel()
	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	rm := result.(TargetModel)
	if !rm.Done() {
		t.Error("should be done after cancel")
	}
	if !rm.Cancelled() {
		t.Error("should be cancelled")
	}
	if rm.Result() != nil {
		t.Error("result should be nil after cancel")
	}
}

func TestTargetViewRenders(t *testing.T) {
	m := NewTargetModel()
	m.width = 80
	v := m.View()
	if !strings.Contains(v, "Step 2: Target MongoDB") {
		t.Error("view should contain title")
	}
	if !strings.Contains(v, "Connection String") {
		t.Error("view should contain Connection String label")
	}
	if !strings.Contains(v, "Database") {
		t.Error("view should contain Database label")
	}
	if !strings.Contains(v, "Press Enter on Database") {
		t.Error("view should contain help text")
	}
}

func TestTargetViewShowsSpinner(t *testing.T) {
	m := NewTargetModel()
	m.connecting = true
	v := m.View()
	if !strings.Contains(v, "Testing connection") {
		t.Error("view should show connecting status")
	}
}

func TestTargetViewShowsError(t *testing.T) {
	m := NewTargetModel()
	m.err = fmt.Errorf("test error")
	m.statusMsg = "Connection failed: test error"
	v := m.View()
	if !strings.Contains(v, "Connection failed") {
		t.Error("view should show error message")
	}
	if !strings.Contains(v, "Fix the issue") {
		t.Error("view should show retry hint")
	}
}

func TestTargetBuildConfigDefaults(t *testing.T) {
	m := NewTargetModel()
	cfg := m.buildConfig()
	if cfg.Type != "mongodb" {
		t.Errorf("expected type 'mongodb', got %q", cfg.Type)
	}
	if cfg.ConnectionString != "mongodb://localhost:27017" {
		t.Errorf("expected default connection string, got %q", cfg.ConnectionString)
	}
	if cfg.Database != "mydb" {
		t.Errorf("expected default database 'mydb', got %q", cfg.Database)
	}
}

func TestTargetBuildConfigCustom(t *testing.T) {
	m := NewTargetModel()
	m.inputs[targetFieldConnStr].SetValue("mongodb://prod:27017")
	m.inputs[targetFieldDatabase].SetValue("production")
	cfg := m.buildConfig()
	if cfg.ConnectionString != "mongodb://prod:27017" {
		t.Errorf("expected custom connection string, got %q", cfg.ConnectionString)
	}
	if cfg.Database != "production" {
		t.Errorf("expected database 'production', got %q", cfg.Database)
	}
}

func TestTargetConnectDoneSuccess(t *testing.T) {
	m := NewTargetModel()
	m.connecting = true
	result, _ := m.Update(targetConnectDoneMsg{})
	rm := result.(TargetModel)
	if !rm.Done() {
		t.Error("should be done after success")
	}
	if rm.Cancelled() {
		t.Error("should not be cancelled")
	}
	if rm.Result() == nil {
		t.Error("result should not be nil after success")
	}
	if rm.connecting {
		t.Error("should not be connecting after done msg")
	}
}

func TestTargetConnectDoneError(t *testing.T) {
	m := NewTargetModel()
	m.connecting = true
	result, _ := m.Update(targetConnectDoneMsg{err: fmt.Errorf("connection refused")})
	rm := result.(TargetModel)
	if rm.Done() {
		t.Error("should not be done after error")
	}
	if rm.connecting {
		t.Error("should not be connecting after error")
	}
	if rm.err == nil {
		t.Error("err should be set")
	}
	if !strings.Contains(rm.statusMsg, "connection refused") {
		t.Errorf("statusMsg should contain error, got %q", rm.statusMsg)
	}
}

func TestTargetEnterOnConnStr_AdvancesFocus(t *testing.T) {
	m := NewTargetModel()
	// Focus is on connStr
	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	rm := result.(TargetModel)
	if rm.focused != targetFieldDatabase {
		t.Errorf("enter on connStr should advance to database, got %d", rm.focused)
	}
}

func TestTargetIgnoresInputWhileConnecting(t *testing.T) {
	m := NewTargetModel()
	m.connecting = true
	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	rm := result.(TargetModel)
	// Focus should not change while connecting
	if rm.focused != targetFieldConnStr {
		t.Errorf("focus should not change while connecting, got %d", rm.focused)
	}
}
