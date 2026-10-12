package service

import (
	"fmt"
	"news-aggregator/internal/models"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockRepository struct {
	mock.Mock
}

func (m *MockRepository) GetUserByKey(apiKey string) (*models.User, error) {
	args := m.Called(apiKey)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockRepository) UpdateUserQuota(userID uint, newQuota int) error {
	args := m.Called(userID, newQuota)
	return args.Error(0)
}

func (m *MockRepository) BatchInsertArticles(articles []models.Article) error {
	args := m.Called(articles)
	return args.Error(0)
}

func (m *MockRepository) GetCachedArticles(url string) ([]models.Article, error) {
	args := m.Called(url)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.Article), args.Error(1)
}

func TestSummarizeArticles_QuotaExceeded(t *testing.T) {
	mockRepo := new(MockRepository)
	svc := NewNewsService(mockRepo)

	user := &models.User{}
	user.Quota = 0
	mockRepo.On("GetUserByKey", "test-key").Return(user, nil)

	urls := []string{"https://example.com/1"}
	res, err := svc.SummarizeArticles("test-key", urls)

	assert.Nil(t, res)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "quota exceeded")
}

func TestSummarizeArticles_InvalidKey(t *testing.T) {
	mockRepo := new(MockRepository)
	svc := NewNewsService(mockRepo)

	mockRepo.On("GetUserByKey", "invalid-key").Return(nil, fmt.Errorf("user not found"))

	urls := []string{"https://example.com/1"}
	res, err := svc.SummarizeArticles("invalid-key", urls)

	assert.Nil(t, res)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "authentication failed")
}
