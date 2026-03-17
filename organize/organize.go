package organize

import (
	"context"
	"fmt"
	"path"
	"strings"

	putio "github.com/putdotio/go-putio"

	"github.com/jack/teleput/config"
)

type Category string

const (
	CategoryMovie     Category = "movie"
	CategoryTV        Category = "tv"
	CategoryMusic     Category = "music"
	CategoryAudiobook Category = "audiobook"
	CategoryBook      Category = "book"
	CategoryJunk      Category = "junk"
	CategoryOther     Category = "other"
)

type FileInfo struct {
	ID          int64
	Name        string
	Size        int64
	ContentType string
	ParentID    int64
	IsDir       bool
}

type Classification struct {
	FileID   int64    `json:"file_id"`
	Category Category `json:"category"`
	NewName  string   `json:"new_name"`
	// TV-specific
	ShowName string `json:"show_name,omitempty"`
	Season   int    `json:"season,omitempty"`
	Episode  int    `json:"episode,omitempty"`
	// Music-specific
	Artist string `json:"artist,omitempty"`
	Album  string `json:"album,omitempty"`
}

type ActionType string

const (
	ActionCreateFolder ActionType = "create_folder"
	ActionRename       ActionType = "rename"
	ActionMove         ActionType = "move"
	ActionDelete       ActionType = "delete"
)

type Action struct {
	Type     ActionType
	FileID   int64
	FileName string
	ParentID int64
	NewName  string
	DestPath string
}

func (a Action) Description() string {
	switch a.Type {
	case ActionCreateFolder:
		return fmt.Sprintf("Create folder %q", a.NewName)
	case ActionRename:
		return fmt.Sprintf("%q → %q", a.FileName, a.NewName)
	case ActionMove:
		return fmt.Sprintf("Move %q to %s", a.FileName, a.DestPath)
	case ActionDelete:
		return fmt.Sprintf("Delete %q", a.FileName)
	default:
		return ""
	}
}

type Plan struct {
	Actions      []Action
	CreateCount  int
	RenameCount  int
	MoveCount    int
	DeleteCount  int
	SourceFolder string
}

type Result struct {
	Completed int
	Total     int
	Current   string
	Err       error
}

type PathConfig struct {
	Paths      map[Category]int64
	DeleteJunk bool
}

func ResolvePathConfig(ctx context.Context, client *putio.Client, cfg config.OrganizerConfig) (*PathConfig, error) {
	pc := &PathConfig{
		Paths:      make(map[Category]int64),
		DeleteJunk: cfg.DeleteJunk,
	}

	resolve := func(cat Category, p string) error {
		if p == "" {
			return nil
		}
		id, err := ResolvePath(ctx, client, p)
		if err != nil {
			return fmt.Errorf("resolving %s path %q: %w", cat, p, err)
		}
		pc.Paths[cat] = id
		return nil
	}

	if err := resolve(CategoryMovie, cfg.Paths.Movies); err != nil {
		return nil, err
	}
	if err := resolve(CategoryTV, cfg.Paths.TV); err != nil {
		return nil, err
	}
	if err := resolve(CategoryMusic, cfg.Paths.Music); err != nil {
		return nil, err
	}
	if err := resolve(CategoryAudiobook, cfg.Paths.Audiobooks); err != nil {
		return nil, err
	}
	if err := resolve(CategoryBook, cfg.Paths.Books); err != nil {
		return nil, err
	}

	return pc, nil
}

// ResolvePath walks a put.io path like "/Movies/Action" from root, creating
// folders as needed. Returns the final folder ID.
func ResolvePath(ctx context.Context, client *putio.Client, p string) (int64, error) {
	p = strings.Trim(p, "/")
	if p == "" {
		return 0, nil
	}

	parts := strings.Split(p, "/")
	parentID := int64(0)

	for _, part := range parts {
		children, _, err := client.Files.List(ctx, parentID)
		if err != nil {
			return 0, fmt.Errorf("listing folder %d: %w", parentID, err)
		}

		found := false
		for _, f := range children {
			if f.IsDir() && strings.EqualFold(f.Name, part) {
				parentID = f.ID
				found = true
				break
			}
		}

		if !found {
			entry, err := client.Files.CreateFolder(ctx, part, parentID)
			if err != nil {
				return 0, fmt.Errorf("creating folder %q in %d: %w", part, parentID, err)
			}
			parentID = entry.ID
		}
	}

	return parentID, nil
}

// CollectFiles returns a flat list of files in the given folder (non-recursive,
// skipping subdirectories).
func CollectFiles(ctx context.Context, client *putio.Client, folderID int64) ([]FileInfo, error) {
	children, _, err := client.Files.List(ctx, folderID)
	if err != nil {
		return nil, fmt.Errorf("listing folder %d: %w", folderID, err)
	}

	var files []FileInfo
	for _, f := range children {
		files = append(files, FileInfo{
			ID:          f.ID,
			Name:        f.Name,
			Size:        f.Size,
			ContentType: f.ContentType,
			ParentID:    folderID,
			IsDir:       f.IsDir(),
		})
	}
	return files, nil
}

// BuildPlan creates an ordered list of actions from classifications and path config.
func BuildPlan(classifications []Classification, pathCfg *PathConfig, sourceFolder string) Plan {
	plan := Plan{SourceFolder: sourceFolder}

	// Track which destination folders we need to create sub-folders in (e.g. TV show/season)
	neededFolders := make(map[string]bool)

	for _, c := range classifications {
		if c.Category == CategoryJunk {
			if pathCfg.DeleteJunk {
				plan.Actions = append(plan.Actions, Action{
					Type:     ActionDelete,
					FileID:   c.FileID,
					FileName: c.NewName,
				})
				plan.DeleteCount++
			}
			continue
		}

		if c.Category == CategoryOther {
			continue
		}

		destID, hasPath := pathCfg.Paths[c.Category]
		if !hasPath {
			continue
		}

		// Rename if the name changed
		if c.NewName != "" {
			plan.Actions = append(plan.Actions, Action{
				Type:     ActionRename,
				FileID:   c.FileID,
				FileName: fileNameFromClassification(c),
				NewName:  c.NewName,
			})
			plan.RenameCount++
		}

		// For TV shows, create show/season subfolder structure
		destPath := categoryDestPath(c)
		if destPath != "" {
			fullPath := destPath
			if !neededFolders[fullPath] {
				neededFolders[fullPath] = true
				plan.Actions = append(plan.Actions, Action{
					Type:     ActionCreateFolder,
					ParentID: destID,
					NewName:  destPath,
					DestPath: destPath,
				})
				plan.CreateCount++
			}
		}

		// Move to destination
		moveName := c.NewName
		if moveName == "" {
			moveName = fileNameFromClassification(c)
		}
		moveDestPath := string(c.Category)
		if destPath != "" {
			moveDestPath = path.Join(moveDestPath, destPath)
		}
		plan.Actions = append(plan.Actions, Action{
			Type:     ActionMove,
			FileID:   c.FileID,
			FileName: moveName,
			ParentID: destID,
			DestPath: moveDestPath,
		})
		plan.MoveCount++
	}

	return plan
}

func fileNameFromClassification(c Classification) string {
	if c.NewName != "" {
		return c.NewName
	}
	return fmt.Sprintf("file-%d", c.FileID)
}

func categoryDestPath(c Classification) string {
	switch c.Category {
	case CategoryTV:
		if c.ShowName != "" && c.Season > 0 {
			return fmt.Sprintf("%s/Season %02d", c.ShowName, c.Season)
		}
		if c.ShowName != "" {
			return c.ShowName
		}
	case CategoryMusic:
		if c.Artist != "" && c.Album != "" {
			return fmt.Sprintf("%s/%s", c.Artist, c.Album)
		}
		if c.Artist != "" {
			return c.Artist
		}
	case CategoryAudiobook:
		if c.NewName != "" {
			// For audiobooks the subfolder is "Author - Title"
			parts := strings.SplitN(c.NewName, "/", 2)
			if len(parts) > 1 {
				return parts[0]
			}
		}
	}
	return ""
}
