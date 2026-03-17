package organize

import (
	"context"
	"fmt"
	"strings"

	putio "github.com/putdotio/go-putio"
)

// Execute runs a plan's actions sequentially against the put.io API.
// It sends Result messages to the channel after each action.
func Execute(ctx context.Context, client *putio.Client, plan Plan, pathCfg *PathConfig, ch chan<- Result) {
	total := len(plan.Actions)
	// Track created subfolder IDs: "category/subfolder" -> folder ID
	createdFolders := make(map[string]int64)

	for i, action := range plan.Actions {
		result := Result{
			Completed: i + 1,
			Total:     total,
			Current:   action.Description(),
		}

		var err error
		switch action.Type {
		case ActionCreateFolder:
			err = executeCreateFolder(ctx, client, action, pathCfg, createdFolders)
		case ActionRename:
			err = executeRename(ctx, client, action)
		case ActionMove:
			err = executeMove(ctx, client, action, pathCfg, createdFolders)
		case ActionDelete:
			err = executeDelete(ctx, client, action)
		}

		if err != nil {
			result.Err = err
		}

		ch <- result

		if err != nil {
			return
		}
	}
}

func executeCreateFolder(ctx context.Context, client *putio.Client, action Action, pathCfg *PathConfig, created map[string]int64) error {
	// Walk/create nested path under the parent
	parts := strings.Split(action.NewName, "/")
	parentID := action.ParentID

	currentPath := ""
	for _, part := range parts {
		if currentPath == "" {
			currentPath = part
		} else {
			currentPath = currentPath + "/" + part
		}

		// Check if we already created this subfolder
		if id, ok := created[currentPath]; ok {
			parentID = id
			continue
		}

		// Check if it exists
		children, _, err := client.Files.List(ctx, parentID)
		if err != nil {
			return fmt.Errorf("listing folder %d: %w", parentID, err)
		}

		found := false
		for _, f := range children {
			if f.IsDir() && strings.EqualFold(f.Name, part) {
				parentID = f.ID
				created[currentPath] = f.ID
				found = true
				break
			}
		}

		if !found {
			entry, err := client.Files.CreateFolder(ctx, part, parentID)
			if err != nil {
				return fmt.Errorf("creating folder %q: %w", part, err)
			}
			parentID = entry.ID
			created[currentPath] = entry.ID
		}
	}

	return nil
}

func executeRename(ctx context.Context, client *putio.Client, action Action) error {
	err := client.Files.Rename(ctx, action.FileID, action.NewName)
	if err != nil {
		return fmt.Errorf("renaming file %d to %q: %w", action.FileID, action.NewName, err)
	}
	return nil
}

func executeMove(ctx context.Context, client *putio.Client, action Action, pathCfg *PathConfig, created map[string]int64) error {
	destID := action.ParentID

	// If there's a subfolder path (e.g. "Show Name/Season 01"), use the created folder
	if action.DestPath != "" {
		// Strip the category prefix to get the subfolder key
		parts := strings.SplitN(action.DestPath, "/", 2)
		if len(parts) > 1 {
			subPath := parts[1]
			if id, ok := created[subPath]; ok {
				destID = id
			}
		}
	}

	err := client.Files.Move(ctx, destID, action.FileID)
	if err != nil {
		return fmt.Errorf("moving file %d to folder %d: %w", action.FileID, destID, err)
	}
	return nil
}

func executeDelete(ctx context.Context, client *putio.Client, action Action) error {
	err := client.Files.Delete(ctx, action.FileID)
	if err != nil {
		return fmt.Errorf("deleting file %d: %w", action.FileID, err)
	}
	return nil
}
