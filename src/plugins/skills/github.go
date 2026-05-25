package skills

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const (
	githubRepo    = "thinktwiceco/agent-forge"
	githubBaseURL = "https://api.github.com"
	githubBranch  = "main"
)

type githubGetFunc func(path string, out any) error

var githubGet githubGetFunc = defaultGithubGet

type ghContentsEntry struct {
	Name     string `json:"name"`
	Path     string `json:"path"`
	Type     string `json:"type"`
	SHA      string `json:"sha"`
	Content  string `json:"content"`
	Encoding string `json:"encoding"`
}

type ghTreeEntry struct {
	Path string `json:"path"`
	Type string `json:"type"`
	SHA  string `json:"sha"`
}

type ghTreeResponse struct {
	Tree      []ghTreeEntry `json:"tree"`
	Truncated bool          `json:"truncated"`
}

type ghBlobResponse struct {
	Content  string `json:"content"`
	Encoding string `json:"encoding"`
}

type installableSkill struct {
	Slug        string
	Name        string
	Description string
	Version     string
	Usage       string
}

func defaultGithubGet(path string, out any) error {
	url := fmt.Sprintf("%s/%s", githubBaseURL, path)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read body failed: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("GitHub API returned %d: %s", resp.StatusCode, string(body))
	}

	return json.Unmarshal(body, out)
}

func fetchInstallableSkills() ([]installableSkill, error) {
	var entries []ghContentsEntry
	if err := githubGet(fmt.Sprintf("repos/%s/contents/%s?ref=%s", githubRepo, remoteSkillsRoot, githubBranch), &entries); err != nil {
		return nil, fmt.Errorf("failed to list remote skills: %w", err)
	}

	result := make([]installableSkill, 0, len(entries))
	for _, entry := range entries {
		if entry.Type != "dir" {
			continue
		}

		skill, err := fetchRemoteSkill(entry.Name)
		if err != nil {
			result = append(result, installableSkill{
				Slug: entry.Name,
				Name: entry.Name,
			})
			continue
		}
		result = append(result, installableSkill{
			Slug:        entry.Name,
			Name:        skill.Name,
			Description: skill.Description,
			Version:     skill.Version,
			Usage:       skill.Usage,
		})
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Slug < result[j].Slug
	})
	return result, nil
}

func fetchRemoteSkill(slug string) (*Skill, error) {
	var entry ghContentsEntry
	path := fmt.Sprintf("repos/%s/contents/%s/%s/%s?ref=%s", githubRepo, remoteSkillsRoot, slug, skillFileName, githubBranch)
	if err := githubGet(path, &entry); err != nil {
		return nil, err
	}

	data, err := decodeContent(entry.Content, entry.Encoding)
	if err != nil {
		return nil, fmt.Errorf("decode remote skill: %w", err)
	}

	return loadSkillFromBytes(filepath.Join(remoteSkillsRoot, slug), data)
}

func installRemoteSkill(slug, destRoot string) (*Skill, error) {
	tmpRoot, err := os.MkdirTemp("", "agent-forge-skill-*")
	if err != nil {
		return nil, err
	}
	defer func() { _ = os.RemoveAll(tmpRoot) }()

	sourceDir, err := downloadRemoteSkill(slug, tmpRoot)
	if err != nil {
		return nil, err
	}
	return installLocalSkillFromDir(sourceDir, destRoot)
}

func downloadRemoteSkill(slug, destRoot string) (string, error) {
	var treeResp ghTreeResponse
	if err := githubGet(fmt.Sprintf("repos/%s/git/trees/%s?recursive=1", githubRepo, githubBranch), &treeResp); err != nil {
		return "", fmt.Errorf("failed to fetch repo tree: %w", err)
	}

	prefix := fmt.Sprintf("%s/%s/", remoteSkillsRoot, slug)
	var blobs []ghTreeEntry
	for _, entry := range treeResp.Tree {
		if entry.Type == "blob" && strings.HasPrefix(entry.Path, prefix) {
			blobs = append(blobs, entry)
		}
	}
	if len(blobs) == 0 {
		return "", fmt.Errorf("skill '%s' not found in remote repo (no files under %s/%s/)", slug, remoteSkillsRoot, slug)
	}

	for _, blob := range blobs {
		relPath := strings.TrimPrefix(blob.Path, remoteSkillsRoot+"/")
		destPath := filepath.Join(destRoot, filepath.FromSlash(relPath))
		if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
			return "", fmt.Errorf("mkdir %s: %w", filepath.Dir(destPath), err)
		}

		content, err := fetchBlob(blob.SHA)
		if err != nil {
			return "", fmt.Errorf("fetch blob %s (%s): %w", blob.Path, blob.SHA, err)
		}
		if err := os.WriteFile(destPath, content, 0644); err != nil {
			return "", fmt.Errorf("write %s: %w", destPath, err)
		}
	}

	return filepath.Join(destRoot, slug), nil
}

func fetchBlob(sha string) ([]byte, error) {
	var blob ghBlobResponse
	if err := githubGet(fmt.Sprintf("repos/%s/git/blobs/%s", githubRepo, sha), &blob); err != nil {
		return nil, err
	}
	return decodeContent(blob.Content, blob.Encoding)
}

func decodeContent(content, encoding string) ([]byte, error) {
	if encoding != "base64" {
		return []byte(content), nil
	}
	clean := strings.ReplaceAll(content, "\n", "")
	return base64.StdEncoding.DecodeString(clean)
}
