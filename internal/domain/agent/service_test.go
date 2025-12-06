package agent

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"prometheus/pkg/errors"
)

// MockRepository is a mock implementation of agent.Repository
type MockRepository struct {
	mock.Mock
}

func (m *MockRepository) Create(ctx context.Context, a *Agent) error {
	args := m.Called(ctx, a)
	return args.Error(0)
}

func (m *MockRepository) GetByID(ctx context.Context, id int) (*Agent, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Agent), args.Error(1)
}

func (m *MockRepository) GetByIdentifier(ctx context.Context, identifier string) (*Agent, error) {
	args := m.Called(ctx, identifier)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Agent), args.Error(1)
}

func (m *MockRepository) FindOrCreate(ctx context.Context, a *Agent) (*Agent, bool, error) {
	args := m.Called(ctx, a)
	if args.Get(0) == nil {
		return nil, args.Bool(1), args.Error(2)
	}
	return args.Get(0).(*Agent), args.Bool(1), args.Error(2)
}

func (m *MockRepository) Update(ctx context.Context, a *Agent) error {
	args := m.Called(ctx, a)
	return args.Error(0)
}

func (m *MockRepository) List(ctx context.Context) ([]*Agent, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*Agent), args.Error(1)
}

func (m *MockRepository) ListActive(ctx context.Context) ([]*Agent, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*Agent), args.Error(1)
}

func (m *MockRepository) ListByCategory(ctx context.Context, category string) ([]*Agent, error) {
	args := m.Called(ctx, category)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*Agent), args.Error(1)
}

func (m *MockRepository) Delete(ctx context.Context, id int) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func TestService_Create(t *testing.T) {
	ctx := context.Background()

	t.Run("creates agent with valid data", func(t *testing.T) {
		repo := new(MockRepository)
		svc := NewService(repo)

		a := &Agent{
			Identifier:   "test_agent",
			Name:         "Test Agent",
			SystemPrompt: "Test prompt",
		}

		repo.On("Create", ctx, a).Return(nil)

		err := svc.Create(ctx, a)
		require.NoError(t, err)
		repo.AssertExpectations(t)
	})

	t.Run("fails with empty identifier", func(t *testing.T) {
		repo := new(MockRepository)
		svc := NewService(repo)

		a := &Agent{
			Identifier:   "", // Empty
			Name:         "Test",
			SystemPrompt: "Prompt",
		}

		err := svc.Create(ctx, a)
		assert.ErrorIs(t, err, errors.ErrInvalidInput)
		repo.AssertNotCalled(t, "Create")
	})

	t.Run("fails with empty name", func(t *testing.T) {
		repo := new(MockRepository)
		svc := NewService(repo)

		a := &Agent{
			Identifier:   "test",
			Name:         "", // Empty
			SystemPrompt: "Prompt",
		}

		err := svc.Create(ctx, a)
		assert.ErrorIs(t, err, errors.ErrInvalidInput)
		repo.AssertNotCalled(t, "Create")
	})

	t.Run("fails with empty system prompt", func(t *testing.T) {
		repo := new(MockRepository)
		svc := NewService(repo)

		a := &Agent{
			Identifier:   "test",
			Name:         "Test",
			SystemPrompt: "", // Empty
		}

		err := svc.Create(ctx, a)
		assert.ErrorIs(t, err, errors.ErrInvalidInput)
		repo.AssertNotCalled(t, "Create")
	})
}

func TestService_GetByIdentifier(t *testing.T) {
	ctx := context.Background()

	t.Run("retrieves agent successfully", func(t *testing.T) {
		repo := new(MockRepository)
		svc := NewService(repo)

		expected := &Agent{
			ID:         1,
			Identifier: "test_agent",
			Name:       "Test Agent",
		}

		repo.On("GetByIdentifier", ctx, "test_agent").Return(expected, nil)

		result, err := svc.GetByIdentifier(ctx, "test_agent")
		require.NoError(t, err)
		assert.Equal(t, expected, result)
		repo.AssertExpectations(t)
	})

	t.Run("returns error for empty identifier", func(t *testing.T) {
		repo := new(MockRepository)
		svc := NewService(repo)

		_, err := svc.GetByIdentifier(ctx, "")
		assert.ErrorIs(t, err, errors.ErrInvalidInput)
		repo.AssertNotCalled(t, "GetByIdentifier")
	})
}

func TestService_Update(t *testing.T) {
	ctx := context.Background()

	t.Run("updates agent successfully", func(t *testing.T) {
		repo := new(MockRepository)
		svc := NewService(repo)

		a := &Agent{
			ID:           1,
			Identifier:   "test",
			Name:         "Updated",
			SystemPrompt: "Updated prompt",
		}

		repo.On("Update", ctx, a).Return(nil)

		err := svc.Update(ctx, a)
		require.NoError(t, err)
		repo.AssertExpectations(t)
	})

	t.Run("fails with nil agent", func(t *testing.T) {
		repo := new(MockRepository)
		svc := NewService(repo)

		err := svc.Update(ctx, nil)
		assert.ErrorIs(t, err, errors.ErrInvalidInput)
		repo.AssertNotCalled(t, "Update")
	})

	t.Run("fails with zero ID", func(t *testing.T) {
		repo := new(MockRepository)
		svc := NewService(repo)

		a := &Agent{
			ID:           0, // Invalid
			SystemPrompt: "Prompt",
		}

		err := svc.Update(ctx, a)
		assert.ErrorIs(t, err, errors.ErrInvalidInput)
		repo.AssertNotCalled(t, "Update")
	})

	t.Run("fails with empty system prompt", func(t *testing.T) {
		repo := new(MockRepository)
		svc := NewService(repo)

		a := &Agent{
			ID:           1,
			SystemPrompt: "", // Empty
		}

		err := svc.Update(ctx, a)
		assert.ErrorIs(t, err, errors.ErrInvalidInput)
		repo.AssertNotCalled(t, "Update")
	})
}

func TestService_FindOrCreate(t *testing.T) {
	ctx := context.Background()

	t.Run("delegates to repository", func(t *testing.T) {
		repo := new(MockRepository)
		svc := NewService(repo)

		a := &Agent{
			Identifier:   "test",
			Name:         "Test",
			SystemPrompt: "Prompt",
		}

		expected := &Agent{ID: 1, Identifier: "test"}
		repo.On("FindOrCreate", ctx, a).Return(expected, true, nil)

		result, wasCreated, err := svc.FindOrCreate(ctx, a)
		require.NoError(t, err)
		assert.True(t, wasCreated)
		assert.Equal(t, expected, result)
		repo.AssertExpectations(t)
	})

	t.Run("validates input", func(t *testing.T) {
		repo := new(MockRepository)
		svc := NewService(repo)

		tests := []struct {
			name  string
			agent *Agent
		}{
			{"nil agent", nil},
			{"empty identifier", &Agent{Identifier: "", Name: "Test", SystemPrompt: "Prompt"}},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				_, _, err := svc.FindOrCreate(ctx, tt.agent)
				assert.ErrorIs(t, err, errors.ErrInvalidInput)
				repo.AssertNotCalled(t, "FindOrCreate")
			})
		}
	})
}

func TestService_ListByCategory(t *testing.T) {
	ctx := context.Background()

	t.Run("lists agents by category", func(t *testing.T) {
		repo := new(MockRepository)
		svc := NewService(repo)

		expected := []*Agent{
			{ID: 1, Identifier: "expert1", Category: CategoryExpert},
			{ID: 2, Identifier: "expert2", Category: CategoryExpert},
		}

		repo.On("ListByCategory", ctx, CategoryExpert).Return(expected, nil)

		result, err := svc.ListByCategory(ctx, CategoryExpert)
		require.NoError(t, err)
		assert.Equal(t, expected, result)
		repo.AssertExpectations(t)
	})

	t.Run("validates empty category", func(t *testing.T) {
		repo := new(MockRepository)
		svc := NewService(repo)

		_, err := svc.ListByCategory(ctx, "")
		assert.ErrorIs(t, err, errors.ErrInvalidInput)
		repo.AssertNotCalled(t, "ListByCategory")
	})
}
