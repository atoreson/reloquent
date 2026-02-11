package aws

import "context"

// MockProvisioner is a test double for the Provisioner interface.
type MockProvisioner struct {
	ProvisionResult   *ProvisionResult
	ProvisionErr      error
	StatusResult      *ProvisionStatus
	StatusErr         error
	SubmitStepErr     error
	TeardownErr       error

	// Track calls
	ProvisionCalled  bool
	ProvisionedPlan  *ProvisionPlan
	StatusCalls      int
	SubmitStepCalls  int
	TeardownCalled   bool
	TeardownResource string
}

func (m *MockProvisioner) Provision(_ context.Context, plan ProvisionPlan) (*ProvisionResult, error) {
	m.ProvisionCalled = true
	m.ProvisionedPlan = &plan
	return m.ProvisionResult, m.ProvisionErr
}

func (m *MockProvisioner) Status(_ context.Context, _ string) (*ProvisionStatus, error) {
	m.StatusCalls++
	return m.StatusResult, m.StatusErr
}

func (m *MockProvisioner) SubmitStep(_ context.Context, _ string, _ string) error {
	m.SubmitStepCalls++
	return m.SubmitStepErr
}

func (m *MockProvisioner) Teardown(_ context.Context, resourceID string) error {
	m.TeardownCalled = true
	m.TeardownResource = resourceID
	return m.TeardownErr
}
