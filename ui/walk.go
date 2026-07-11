package ui

import (
	"context"
	"path/filepath"

	putio "github.com/putdotio/go-putio"
)

type lister interface {
	List(ctx context.Context, parentID int64) ([]putio.File, putio.File, error)
}

type remoteFile struct {
	ID      int64
	RelPath string
	Size    int64
}

// collectFiles flattens a file or folder into downloadable entries with
// paths relative to the selection (rooted at the folder's own name).
func collectFiles(ctx context.Context, l lister, root putio.File) ([]remoteFile, error) {
	if !root.IsDir() {
		return []remoteFile{{ID: root.ID, RelPath: root.Name, Size: root.Size}}, nil
	}

	var out []remoteFile
	type frame struct {
		id     int64
		prefix string
	}
	stack := []frame{{root.ID, root.Name}}
	for len(stack) > 0 {
		f := stack[0]
		stack = stack[1:]
		children, _, err := l.List(ctx, f.id)
		if err != nil {
			return nil, err
		}
		for _, c := range children {
			if c.IsDir() {
				stack = append(stack, frame{c.ID, filepath.Join(f.prefix, c.Name)})
			} else {
				out = append(out, remoteFile{ID: c.ID, RelPath: filepath.Join(f.prefix, c.Name), Size: c.Size})
			}
		}
	}
	return out, nil
}
