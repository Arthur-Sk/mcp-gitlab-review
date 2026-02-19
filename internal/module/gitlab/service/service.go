package service

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	"mcp-gitlab-review/internal/module/gitlab/domain"
	"mcp-gitlab-review/internal/module/gitlab/repository"
	"mcp-gitlab-review/internal/platform/cache"
)

type Service struct {
	client *repository.Client
	cache  *cache.Cache
	logger *zap.Logger
}

func NewService(client *repository.Client, cache *cache.Cache, logger *zap.Logger) *Service {
	return &Service{
		client: client,
		cache:  cache,
		logger: logger,
	}
}

func (s *Service) GetMRSummary(ctx context.Context, projectPath string, mrIID int) (*domain.MergeRequest, error) {
	cacheKey := fmt.Sprintf("mr:%s:%d:summary", projectPath, mrIID)
	if cached, ok := s.cache.Get(cacheKey); ok {
		s.logger.Debug("cache hit for MR summary", zap.String("key", cacheKey))

		return cached.(*domain.MergeRequest), nil
	}

	mr, err := s.client.GetMergeRequest(ctx, projectPath, mrIID)
	if err != nil {
		return nil, fmt.Errorf("fetch MR summary: %w", err)
	}

	commits, err := s.client.GetMergeRequestCommits(ctx, projectPath, mrIID)
	if err != nil {
		s.logger.Warn("failed to fetch commits, continuing without them", zap.Error(err))
	} else {
		messages := make([]string, 0, len(commits))
		for _, c := range commits {
			messages = append(messages, c.Message)
		}
		mr.CommitMessages = messages
	}

	s.cache.Set(cacheKey, mr)

	return mr, nil
}

func (s *Service) GetMRChanges(ctx context.Context, projectPath string, mrIID int) (*domain.MergeRequestChanges, error) {
	cacheKey := fmt.Sprintf("mr:%s:%d:changes", projectPath, mrIID)
	if cached, ok := s.cache.Get(cacheKey); ok {
		s.logger.Debug("cache hit for MR changes", zap.String("key", cacheKey))

		return cached.(*domain.MergeRequestChanges), nil
	}

	changes, err := s.client.GetMergeRequestChanges(ctx, projectPath, mrIID)
	if err != nil {
		return nil, fmt.Errorf("fetch MR changes: %w", err)
	}

	s.cache.Set(cacheKey, changes)

	return changes, nil
}

func (s *Service) GetFileContent(ctx context.Context, projectPath string, ref string, filePath string) (string, error) {
	cacheKey := fmt.Sprintf("file:%s:%s:%s", projectPath, ref, filePath)
	if cached, ok := s.cache.Get(cacheKey); ok {
		s.logger.Debug("cache hit for file content", zap.String("key", cacheKey))

		return cached.(string), nil
	}

	content, err := s.client.GetFileContent(ctx, projectPath, ref, filePath)
	if err != nil {
		return "", fmt.Errorf("fetch file content: %w", err)
	}

	s.cache.Set(cacheKey, content)

	return content, nil
}

func (s *Service) GetRepositoryTree(ctx context.Context, projectPath string, ref string, path string) ([]domain.TreeEntry, error) {
	cacheKey := fmt.Sprintf("tree:%s:%s:%s", projectPath, ref, path)
	if cached, ok := s.cache.Get(cacheKey); ok {
		s.logger.Debug("cache hit for repository tree", zap.String("key", cacheKey))

		return cached.([]domain.TreeEntry), nil
	}

	entries, err := s.client.GetRepositoryTree(ctx, projectPath, ref, path, true)
	if err != nil {
		return nil, fmt.Errorf("fetch repository tree: %w", err)
	}

	s.cache.Set(cacheKey, entries)

	return entries, nil
}

func (s *Service) PostMRComment(ctx context.Context, projectPath string, mrIID int, comment domain.CreateDiscussionRequest) error {
	if err := s.client.CreateMRDiscussion(ctx, projectPath, mrIID, comment); err != nil {
		return fmt.Errorf("post MR comment: %w", err)
	}

	return nil
}
