package api

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ResolverFunc transforms a string value into another string value.
// Used to pre-process body arguments before sending the HTTP request.
type ResolverFunc func(value string) (string, error)

// resolverRegistry maps resolver names to their implementations.
// Add new resolvers here; no handler changes required.
var resolverRegistry = map[string]ResolverFunc{
	"resolve_to_base64": resolveToBase64,
}

// resolveToBase64 reads a local file and returns a base64 data URI.
func resolveToBase64(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("resolve_to_base64: cannot read file %q: %w", path, err)
	}
	mime := mimeFromExt(filepath.Ext(path))
	encoded := base64.StdEncoding.EncodeToString(data)
	return "data:" + mime + ";base64," + encoded, nil
}

func mimeFromExt(ext string) string {
	switch strings.ToLower(ext) {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".gif":
		return "image/gif"
	case ".webp":
		return "image/webp"
	default:
		return "application/octet-stream"
	}
}

// resolveBodyArgs scans bodyMap for keys containing "_$".
// For each such key, it splits on "_$" to get (resolverName, paramName),
// looks up the resolver, calls it, and replaces the entry in-place.
// workingDir is used to resolve relative file paths for resolve_to_base64.
func resolveBodyArgs(bodyMap map[string]any, workingDir string) error {
	for key, val := range bodyMap {
		idx := strings.Index(key, "_$")
		if idx == -1 {
			continue
		}
		resolverName := key[:idx]
		paramName := key[idx+2:]
		if paramName == "" {
			return fmt.Errorf("invalid resolver key %q: missing parameter name after _$", key)
		}
		resolver, ok := resolverRegistry[resolverName]
		if !ok {
			return fmt.Errorf("unknown resolver %q in body key %q", resolverName, key)
		}
		strVal, ok := val.(string)
		if !ok {
			return fmt.Errorf("resolver argument %q must be a string, got %T", key, val)
		}
		// Resolve relative paths for file-based resolvers (e.g. resolve_to_base64)
		if resolverName == "resolve_to_base64" && workingDir != "" && !filepath.IsAbs(strVal) {
			strVal = filepath.Join(workingDir, strVal)
		}
		resolved, err := resolver(strVal)
		if err != nil {
			return err
		}
		delete(bodyMap, key)
		bodyMap[paramName] = resolved
	}
	return nil
}
