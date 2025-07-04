package main

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"
)

func deleteFilesOlderThan(dirPath string, days int) int {
	if days < 0 {
		fmt.Println("CleanupManager: 'days' parameter cannot be negative.")
		return 0
	}

	info, err := os.Stat(dirPath)
	if err != nil || !info.IsDir() {
		fmt.Printf("CleanupManager: Cleanup directory not found or is not a directory: %s\n", dirPath)
		return 0
	}

	cutoff := time.Now().AddDate(0, 0, -days)

	deletedCount := 0

	err = filepath.WalkDir(dirPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			fmt.Printf("CleanupManager: Failed to access %s: %v\n", path, err)
			return nil
		}
		if d.IsDir() {
			return nil
		}

		fileInfo, err := d.Info()
		if err != nil {
			fmt.Printf("CleanupManager: Failed to get info for %s: %v\n", path, err)
			return nil
		}

		if fileInfo.ModTime().Before(cutoff) {
			fmt.Printf("CleanupManager: Deleting old file: %s\n", path)
			if err := os.Remove(path); err == nil {
				deletedCount++
			} else {
				fmt.Printf("CleanupManager: Failed to delete file: %s: %v\n", path, err)
			}
		}
		return nil
	})

	if err != nil {
		fmt.Printf("CleanupManager: Error walking directory: %v\n", err)
	}

	fmt.Printf("CleanupManager: Deletion finished. Successfully deleted %d files older than %d days.\n", deletedCount, days)
	return deletedCount
}
