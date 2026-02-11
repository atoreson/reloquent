package aws

import "context"

// MockClient is a test double for the Client interface.
type MockClient struct {
	Identity     *CallerIdentity
	IdentityErr  error
	EMRAccess    bool
	EMRErr       error
	GlueAccess   bool
	GlueErr      error
	UploadErr    error
	UploadFileErr error
	DeleteErr    error

	// Track calls
	UploadedObjects map[string][]byte // key → data
	UploadedFiles   map[string]string // key → local path
	DeletedPrefixes []string
}

// NewMockClient creates a new MockClient with default values.
func NewMockClient() *MockClient {
	return &MockClient{
		Identity: &CallerIdentity{
			Account: "123456789012",
			ARN:     "arn:aws:iam::123456789012:user/test",
			UserID:  "AIDA12345",
		},
		UploadedObjects: make(map[string][]byte),
		UploadedFiles:   make(map[string]string),
	}
}

func (m *MockClient) VerifyCredentials(_ context.Context) (*CallerIdentity, error) {
	return m.Identity, m.IdentityErr
}

func (m *MockClient) CheckEMRAccess(_ context.Context) (bool, error) {
	return m.EMRAccess, m.EMRErr
}

func (m *MockClient) CheckGlueAccess(_ context.Context) (bool, error) {
	return m.GlueAccess, m.GlueErr
}

func (m *MockClient) UploadToS3(_ context.Context, bucket, key string, data []byte) error {
	if m.UploadErr != nil {
		return m.UploadErr
	}
	fullKey := bucket + "/" + key
	m.UploadedObjects[fullKey] = data
	return nil
}

func (m *MockClient) UploadFileToS3(_ context.Context, bucket, key, localPath string) error {
	if m.UploadFileErr != nil {
		return m.UploadFileErr
	}
	fullKey := bucket + "/" + key
	m.UploadedFiles[fullKey] = localPath
	return nil
}

func (m *MockClient) DeleteS3Prefix(_ context.Context, bucket, prefix string) error {
	if m.DeleteErr != nil {
		return m.DeleteErr
	}
	m.DeletedPrefixes = append(m.DeletedPrefixes, bucket+"/"+prefix)
	return nil
}
