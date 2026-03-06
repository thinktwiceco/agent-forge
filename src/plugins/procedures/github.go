package procedures

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	githubRepo    = "thinktwiceco/agent-forge"
	githubBaseURL = "https://api.github.com"
)

type ghContentsEntry struct {
	Name        string `json:"name"`
	Path        string `json:"path"`
	Type        string `json:"type"` // "file" or "dir"
	SHA         string `json:"sha"`
	DownloadURL string `json:"download_url"`
	Content     string `json:"content"`  // base64, only when fetching a single file
	Encoding    string `json:"encoding"` // "base64"
}

type ghTreeEntry struct {
	Path string `json:"path"`
	Type string `json:"type"` // "blob" or "tree"
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

type remoteProcedure struct {
	Slug        string
	Name        string
	Description string
}

func githubGet(path string, out any) error {
	url := fmt.Sprintf("%s/%s", githubBaseURL, path)
	req, err := http.NewRequest("GET", url, nil)
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

// fetchInstallableProcedures lists all procedures available in the remote
// GitHub repo's procedures/ folder, including their name and description from
// each manifest.yaml.
func fetchInstallableProcedures() ([]remoteProcedure, error) {
	var entries []ghContentsEntry
	if err := githubGet(fmt.Sprintf("repos/%s/contents/procedures", githubRepo), &entries); err != nil {
		return nil, fmt.Errorf("failed to list remote procedures: %w", err)
	}

	var result []remoteProcedure
	for _, e := range entries {
		if e.Type != "dir" {
			continue
		}

		slug := e.Name
		name, desc, err := fetchRemoteManifest(slug)
		if err != nil {
			// include the slug even if we can't read the manifest
			result = append(result, remoteProcedure{Slug: slug, Name: slug})
			continue
		}
		result = append(result, remoteProcedure{Slug: slug, Name: name, Description: desc})
	}

	return result, nil
}

// fetchRemoteManifest fetches and parses the manifest.yaml for a remote procedure slug.
func fetchRemoteManifest(slug string) (name, description string, err error) {
	var entry ghContentsEntry
	path := fmt.Sprintf("repos/%s/contents/procedures/%s/manifest.yaml", githubRepo, slug)
	if err := githubGet(path, &entry); err != nil {
		return "", "", err
	}

	data, err := decodeContent(entry.Content, entry.Encoding)
	if err != nil {
		return "", "", fmt.Errorf("decode manifest: %w", err)
	}

	var m manifest
	if err := yaml.Unmarshal(data, &m); err != nil {
		return "", "", fmt.Errorf("parse manifest: %w", err)
	}

	if m.Name == "" {
		m.Name = slug
	}
	return m.Name, m.Description, nil
}

// installProcedure downloads the full procedure tree for the given slug from
// the remote GitHub repo and writes it into destDir/slug/, preserving paths.
// Returns the number of files written.
func installProcedure(slug, destDir string) (int, error) {
	var treeResp ghTreeResponse
	if err := githubGet(fmt.Sprintf("repos/%s/git/trees/HEAD?recursive=1", githubRepo), &treeResp); err != nil {
		return 0, fmt.Errorf("failed to fetch repo tree: %w", err)
	}

	prefix := fmt.Sprintf("procedures/%s/", slug)
	var blobs []ghTreeEntry
	for _, entry := range treeResp.Tree {
		if entry.Type == "blob" && strings.HasPrefix(entry.Path, prefix) {
			blobs = append(blobs, entry)
		}
	}

	if len(blobs) == 0 {
		return 0, fmt.Errorf("procedure '%s' not found in remote repo (no files under procedures/%s/)", slug, slug)
	}

	written := 0
	for _, blob := range blobs {
		relPath := strings.TrimPrefix(blob.Path, "procedures/")
		destPath := filepath.Join(destDir, relPath)

		if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
			return written, fmt.Errorf("mkdir %s: %w", filepath.Dir(destPath), err)
		}

		content, err := fetchBlob(blob.SHA)
		if err != nil {
			return written, fmt.Errorf("fetch blob %s (%s): %w", blob.Path, blob.SHA, err)
		}

		if err := os.WriteFile(destPath, content, 0644); err != nil {
			return written, fmt.Errorf("write %s: %w", destPath, err)
		}
		written++
	}

	return written, nil
}

// fetchBlob downloads a git blob by SHA and returns its decoded content.
func fetchBlob(sha string) ([]byte, error) {
	var blob ghBlobResponse
	if err := githubGet(fmt.Sprintf("repos/%s/git/blobs/%s", githubRepo, sha), &blob); err != nil {
		return nil, err
	}
	return decodeContent(blob.Content, blob.Encoding)
}

// decodeContent decodes base64-encoded content (with possible embedded newlines).
func decodeContent(content, encoding string) ([]byte, error) {
	if encoding != "base64" {
		return []byte(content), nil
	}
	clean := strings.ReplaceAll(content, "\n", "")
	return base64.StdEncoding.DecodeString(clean)
}
