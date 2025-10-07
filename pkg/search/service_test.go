package search

import (
	"context"
	"testing"

	"github.com/opensearch-project/opensearch-go/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"motion-index-fiber/pkg/search/client"
)

// MockSearchClient implements SearchClient interface for testing
type MockSearchClient struct {
	mock.Mock
}

func (m *MockSearchClient) GetClient() *opensearch.Client {
	args := m.Called()
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(*opensearch.Client)
}

func (m *MockSearchClient) GetIndex() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockSearchClient) IsHealthy() bool {
	args := m.Called()
	return args.Bool(0)
}

func (m *MockSearchClient) Health(ctx context.Context) (*client.HealthStatus, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*client.HealthStatus), args.Error(1)
}

func (m *MockSearchClient) IndexExists(ctx context.Context) (bool, error) {
	args := m.Called(ctx)
	return args.Bool(0), args.Error(1)
}

func (m *MockSearchClient) CreateIndex(ctx context.Context, mapping map[string]interface{}) error {
	args := m.Called(ctx, mapping)
	return args.Error(0)
}

func (m *MockSearchClient) DeleteIndex(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockSearchClient) RefreshIndex(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockSearchClient) Close() error {
	args := m.Called()
	return args.Error(0)
}

func TestNewService(t *testing.T) {
	client := &MockSearchClient{}
	service := NewService(client)

	assert.NotNil(t, service)
}

// TODO: Reimplement comprehensive tests with proper OpenSearch mocking
// The current tests need to be redesigned to work with the service's 
// actual OpenSearch API calls rather than high-level method mocking