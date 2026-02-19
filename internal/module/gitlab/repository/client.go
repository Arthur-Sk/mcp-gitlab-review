package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"go.uber.org/zap"

	"mcp-gitlab-review/internal/module/gitlab/domain"
	"mcp-gitlab-review/internal/platform/config"
)

type Client struct {
	httpClient *http.Client
	apiURL     string
	token      string
	logger     *zap.Logger
}

func NewClient(cfg *config.Config, logger *zap.Logger) *Client {
	return &Client{
		httpClient: &http.Client{},
		apiURL:     cfg.GitLab.APIURL,
		token:      cfg.GitLab.Token,
		logger:     logger,
	}
}

func (c *Client) GetMergeRequest(ctx context.Context, projectPath string, mrIID int) (*domain.MergeRequest, error) {
	endpoint := fmt.Sprintf("/projects/%s/merge_requests/%d", url.PathEscape(projectPath), mrIID)

	var mr domain.MergeRequest
	if err := c.doJSON(ctx, http.MethodGet, endpoint, nil, &mr); err != nil {
		return nil, fmt.Errorf("get merge request: %w", err)
	}

	return &mr, nil
}

func (c *Client) GetMergeRequestCommits(ctx context.Context, projectPath string, mrIID int) ([]domain.Commit, error) {
	endpoint := fmt.Sprintf("/projects/%s/merge_requests/%d/commits", url.PathEscape(projectPath), mrIID)

	var commits []domain.Commit
	if err := c.doJSON(ctx, http.MethodGet, endpoint, nil, &commits); err != nil {
		return nil, fmt.Errorf("get merge request commits: %w", err)
	}

	return commits, nil
}

func (c *Client) GetMergeRequestChanges(ctx context.Context, projectPath string, mrIID int) (*domain.MergeRequestChanges, error) {
	endpoint := fmt.Sprintf("/projects/%s/merge_requests/%d/changes", url.PathEscape(projectPath), mrIID)

	var changes domain.MergeRequestChanges
	if err := c.doJSON(ctx, http.MethodGet, endpoint, nil, &changes); err != nil {
		return nil, fmt.Errorf("get merge request changes: %w", err)
	}

	return &changes, nil
}

func (c *Client) GetFileContent(ctx context.Context, projectPath string, ref string, filePath string) (string, error) {
	endpoint := fmt.Sprintf(
		"/projects/%s/repository/files/%s/raw?ref=%s",
		url.PathEscape(projectPath),
		url.PathEscape(filePath),
		url.QueryEscape(ref),
	)

	body, err := c.doRaw(ctx, http.MethodGet, endpoint)
	if err != nil {
		return "", fmt.Errorf("get file content: %w", err)
	}

	return body, nil
}

func (c *Client) GetRepositoryTree(ctx context.Context, projectPath string, ref string, path string, recursive bool) ([]domain.TreeEntry, error) {
	var allEntries []domain.TreeEntry
	page := 1
	perPage := 100

	for {
		endpoint := fmt.Sprintf(
			"/projects/%s/repository/tree?ref=%s&per_page=%d&page=%d&recursive=%t",
			url.PathEscape(projectPath),
			url.QueryEscape(ref),
			perPage,
			page,
			recursive,
		)

		if path != "" {
			endpoint += "&path=" + url.QueryEscape(path)
		}

		var entries []domain.TreeEntry
		if err := c.doJSON(ctx, http.MethodGet, endpoint, nil, &entries); err != nil {
			return nil, fmt.Errorf("get repository tree page %d: %w", page, err)
		}

		allEntries = append(allEntries, entries...)

		if len(entries) < perPage {
			break
		}

		page++
	}

	return allEntries, nil
}

func (c *Client) CreateMRDiscussion(ctx context.Context, projectPath string, mrIID int, discussion domain.CreateDiscussionRequest) error {
	endpoint := fmt.Sprintf("/projects/%s/merge_requests/%d/discussions", url.PathEscape(projectPath), mrIID)

	if err := c.doJSON(ctx, http.MethodPost, endpoint, discussion, nil); err != nil {
		return fmt.Errorf("create MR discussion: %w", err)
	}

	return nil
}

func (c *Client) doJSON(ctx context.Context, method string, endpoint string, reqBody interface{}, result interface{}) error {
	fullURL := c.apiURL + endpoint

	var bodyReader io.Reader
	if reqBody != nil {
		bodyBytes, err := json.Marshal(reqBody)
		if err != nil {
			return fmt.Errorf("marshal request body: %w", err)
		}

		bodyReader = io.NopCloser(jsonReader(bodyBytes))
	}

	req, err := http.NewRequestWithContext(ctx, method, fullURL, bodyReader)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	c.setHeaders(req)
	if reqBody != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	if err := checkResponse(resp); err != nil {
		return err
	}

	if result != nil {
		if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
			return fmt.Errorf("decode response: %w", err)
		}
	}

	return nil
}

func (c *Client) doRaw(ctx context.Context, method string, endpoint string) (string, error) {
	fullURL := c.apiURL + endpoint

	req, err := http.NewRequestWithContext(ctx, method, fullURL, nil)
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}

	c.setHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	if err := checkResponse(resp); err != nil {
		return "", err
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response body: %w", err)
	}

	return string(body), nil
}

func (c *Client) setHeaders(req *http.Request) {
	req.Header.Set("PRIVATE-TOKEN", c.token)
}

func checkResponse(resp *http.Response) error {
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}

	body, _ := io.ReadAll(resp.Body)

	return fmt.Errorf("GitLab API error (status %d): %s", resp.StatusCode, string(body))
}

func jsonReader(data []byte) io.Reader {
	return &jsonByteReader{data: data}
}

type jsonByteReader struct {
	data   []byte
	offset int
}

func (r *jsonByteReader) Read(p []byte) (int, error) {
	if r.offset >= len(r.data) {
		return 0, io.EOF
	}

	n := copy(p, r.data[r.offset:])
	r.offset += n

	return n, nil
}
