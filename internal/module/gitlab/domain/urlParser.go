package domain

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

type MRReference struct {
	ProjectPath string
	MRIID       int
}

func ParseMRURL(mrURL string) (*MRReference, error) {
	u, err := url.Parse(mrURL)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}

	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) < 4 {
		return nil, fmt.Errorf("invalid MR URL: path too short")
	}

	mrIdx := findMergeRequestsIndex(parts)
	if mrIdx < 0 {
		return nil, fmt.Errorf("invalid MR URL: 'merge_requests' segment not found")
	}

	if mrIdx+1 >= len(parts) {
		return nil, fmt.Errorf("invalid MR URL: missing MR IID after 'merge_requests'")
	}

	iid, err := strconv.Atoi(parts[mrIdx+1])
	if err != nil {
		return nil, fmt.Errorf("invalid MR IID %q: %w", parts[mrIdx+1], err)
	}

	projectPath := extractProjectPath(parts, mrIdx)
	if projectPath == "" {
		return nil, fmt.Errorf("invalid MR URL: could not extract project path")
	}

	return &MRReference{
		ProjectPath: projectPath,
		MRIID:       iid,
	}, nil
}

func findMergeRequestsIndex(parts []string) int {
	for i, p := range parts {
		if p == "merge_requests" {
			return i
		}
	}

	return -1
}

func extractProjectPath(parts []string, mrIdx int) string {
	endIdx := mrIdx
	if endIdx > 0 && parts[endIdx-1] == "-" {
		endIdx--
	}

	if endIdx <= 0 {
		return ""
	}

	return strings.Join(parts[:endIdx], "/")
}
