package fs

import "fmt"

// fileOperationResponse represents a file operation result.
type fileOperationResponse struct {
	Operation    string
	RelativePath string
	AbsolutePath string
	Size         int64
	Modified     string
	Directory    string
	Content      string
	Status       string
	LastModified string
}

// String formats the file operation response as a string.
func (r *fileOperationResponse) String() string {
	switch r.Operation {
	case "Read":
		return fmt.Sprintf(`File Operation: Read
Path (relative): %s
Path (absolute): %s
Size: %d bytes
Modified: %s
Content:
---
%s
---`, r.RelativePath, r.AbsolutePath, r.Size, r.Modified, r.Content)
	case "Write":
		return fmt.Sprintf(`File Operation: Write (%s)
Path (relative): %s
Path (absolute): %s
Size: %d bytes
Modified: %s
Directory: %s`, r.Status, r.RelativePath, r.AbsolutePath, r.Size, r.Modified, r.Directory)
	case "Delete":
		return fmt.Sprintf(`File Operation: Delete
Path (relative): %s
Path (absolute): %s
Size: %d bytes
Last Modified: %s
Status: Successfully deleted`, r.RelativePath, r.AbsolutePath, r.Size, r.LastModified)
	default:
		return fmt.Sprintf("Unknown operation: %s", r.Operation)
	}
}

// rootDirectoryResponse represents a root directory information result.
type rootDirectoryResponse struct {
	AbsolutePath string
	RelativePath string
	IsDirectory  bool
	Permissions  string
}

// String formats the root directory response as a string.
func (r *rootDirectoryResponse) String() string {
	return fmt.Sprintf(`Sandbox Root Directory:
Path (absolute): %s
Path (relative): %s
Exists: true
Is Directory: %v
Permissions: %s`, r.AbsolutePath, r.RelativePath, r.IsDirectory, r.Permissions)
}

// fileEntry represents a single file or directory entry in a listing.
type fileEntry struct {
	Name     string
	Type     string
	Size     int64
	Modified string
	IsDir    bool
}

// listFilesResponse represents a directory listing result.
type listFilesResponse struct {
	Directory    string
	RelativePath string
	AbsolutePath string
	Entries      []fileEntry
	Count        int
}

// String formats the list files response as a string.
func (r *listFilesResponse) String() string {
	result := fmt.Sprintf(`File Operation: List
Directory (relative): %s
Directory (absolute): %s
Total entries: %d

`, r.RelativePath, r.AbsolutePath, r.Count)

	if len(r.Entries) == 0 {
		result += "Directory is empty.\n"
	} else {
		for _, entry := range r.Entries {
			result += fmt.Sprintf("%s  %s  %10d bytes  %s\n",
				entry.Type,
				entry.Name,
				entry.Size,
				entry.Modified,
			)
		}
	}

	return result
}

// fileInfoResponse represents detailed file information result.
type fileInfoResponse struct {
	RelativePath string
	AbsolutePath string
	Size         int64
	Permissions  string
	Mode         string
	IsDirectory  bool
	Modified     string
	UID          uint32
	GID          uint32
}

// String formats the file info response as a string.
func (r *fileInfoResponse) String() string {
	return fmt.Sprintf(`File Information:
Path (relative): %s
Path (absolute): %s
Size: %d bytes
Permissions: %s
Mode: %s
Is Directory: %v
Last Modified: %s
Owner UID: %d
Owner GID: %d`, r.RelativePath, r.AbsolutePath, r.Size, r.Permissions, r.Mode, r.IsDirectory, r.Modified, r.UID, r.GID)
}

// ripgrepResponse represents a ripgrep search result.
type ripgrepResponse struct {
	Pattern      string
	RelativePath string
	AbsolutePath string
	Flags        []string
	MatchCount   int
	Output       string
	Status       string
}

// String formats the ripgrep response as a string.
func (r *ripgrepResponse) String() string {
	result := fmt.Sprintf(`File Operation: Ripgrep Search
Pattern: %s
Path (relative): %s
Path (absolute): %s`, r.Pattern, r.RelativePath, r.AbsolutePath)

	if len(r.Flags) > 0 {
		result += fmt.Sprintf("\nFlags: %s", fmt.Sprintf("%v", r.Flags))
	}

	result += fmt.Sprintf("\n%s\n", r.Status)

	if r.Output != "" {
		result += fmt.Sprintf("\nMatches:\n---\n%s\n---", r.Output)
	}

	return result
}
