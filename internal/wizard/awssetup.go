package wizard

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/reloquent/reloquent/internal/aws"
)

// AWSSetupResult holds the output of the AWS setup step.
type AWSSetupResult struct {
	Region   string
	Profile  string
	S3Bucket string
	Platform string // "emr" or "glue"
	Identity *aws.CallerIdentity
	Access   *aws.PlatformAccess
}

const (
	awsFieldRegion = iota
	awsFieldProfile
	awsFieldS3Bucket
	awsFieldCount
)

// AWSSetupModel is the bubbletea model for Step 7: AWS Setup.
type AWSSetupModel struct {
	inputs       []textinput.Model
	focused      int
	platform     int // 0=auto, 1=EMR, 2=Glue, 3=scripts-only
	identity     *aws.CallerIdentity
	access       *aws.PlatformAccess
	credStatus   string
	done         bool
	cancelled    bool
	width        int
	height       int
}

// NewAWSSetupModel creates an AWS setup model.
func NewAWSSetupModel() AWSSetupModel {
	inputs := make([]textinput.Model, awsFieldCount)

	inputs[awsFieldRegion] = textinput.New()
	inputs[awsFieldRegion].Placeholder = "us-east-1"
	inputs[awsFieldRegion].CharLimit = 20

	inputs[awsFieldProfile] = textinput.New()
	inputs[awsFieldProfile].Placeholder = "default"
	inputs[awsFieldProfile].CharLimit = 50

	inputs[awsFieldS3Bucket] = textinput.New()
	inputs[awsFieldS3Bucket].Placeholder = "my-migration-bucket"
	inputs[awsFieldS3Bucket].CharLimit = 63

	inputs[awsFieldRegion].Focus()

	return AWSSetupModel{
		inputs:     inputs,
		credStatus: "Not checked",
		width:      100,
		height:     24,
	}
}

func (m AWSSetupModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m AWSSetupModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
		case "tab", "down":
			m.inputs[m.focused].Blur()
			m.focused = (m.focused + 1) % awsFieldCount
			m.inputs[m.focused].Focus()
			return m, nil
		case "shift+tab", "up":
			m.inputs[m.focused].Blur()
			m.focused = (m.focused - 1 + awsFieldCount) % awsFieldCount
			m.inputs[m.focused].Focus()
			return m, nil
		case "ctrl+p":
			m.platform = (m.platform + 1) % 4
			return m, nil
		}
	}

	// Update focused input
	var cmd tea.Cmd
	m.inputs[m.focused], cmd = m.inputs[m.focused].Update(msg)
	return m, cmd
}

func (m AWSSetupModel) View() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("Step 7: AWS Setup"))
	b.WriteString("\n\n")

	// Credential status
	b.WriteString(fmt.Sprintf("  Credentials: %s\n", m.credStatus))
	if m.identity != nil {
		b.WriteString(fmt.Sprintf("  Account: %s  ARN: %s\n", m.identity.Account, m.identity.ARN))
	}
	if m.access != nil {
		b.WriteString(fmt.Sprintf("  %s\n", m.access.Message))
	}
	b.WriteString("\n")

	// Input fields
	labels := []string{"Region", "Profile", "S3 Bucket"}
	for i, label := range labels {
		cursor := "  "
		if i == m.focused {
			cursor = "> "
		}
		b.WriteString(fmt.Sprintf("%s%-10s %s\n", cursor, label+":", m.inputs[i].View()))
	}

	// Platform choice
	b.WriteString("\n")
	platforms := []string{"Auto", "EMR", "Glue", "Scripts Only"}
	b.WriteString("  Platform: ")
	for i, p := range platforms {
		if i == m.platform {
			b.WriteString(highlightStyle.Render("[" + p + "]"))
		} else {
			b.WriteString(dimStyle.Render(" " + p + " "))
		}
		b.WriteString(" ")
	}
	b.WriteString("\n")

	b.WriteString("\n")
	b.WriteString(dimStyle.Render("  tab: next field  ctrl+p: cycle platform  enter: continue  q: cancel"))

	return b.String()
}

// Done returns true when the model is finished.
func (m AWSSetupModel) Done() bool {
	return m.done
}

// Cancelled returns true if the user cancelled.
func (m AWSSetupModel) Cancelled() bool {
	return m.cancelled
}

// Result returns the AWS setup result.
func (m AWSSetupModel) Result() *AWSSetupResult {
	platforms := []string{"auto", "emr", "glue", "scripts-only"}
	return &AWSSetupResult{
		Region:   m.inputs[awsFieldRegion].Value(),
		Profile:  m.inputs[awsFieldProfile].Value(),
		S3Bucket: m.inputs[awsFieldS3Bucket].Value(),
		Platform: platforms[m.platform],
		Identity: m.identity,
		Access:   m.access,
	}
}

// SetCredentialResult updates the credential check status.
func (m *AWSSetupModel) SetCredentialResult(identity *aws.CallerIdentity, access *aws.PlatformAccess) {
	m.identity = identity
	m.access = access
	if identity != nil {
		m.credStatus = successStyle.Render("Verified")
	} else {
		m.credStatus = errStyle.Render("Not configured")
	}
}
