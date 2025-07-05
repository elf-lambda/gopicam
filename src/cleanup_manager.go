package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

func deleteFoldersOlderThan(dirPath string, days int) int {
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

	entries, err := os.ReadDir(dirPath)
	if err != nil {
		fmt.Printf("CleanupManager: Failed to read directory: %v\n", err)
		return 0
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		folderName := entry.Name()

		// Parse folder name as date: YYYYMMDD
		folderDate, err := time.Parse("20060102", folderName)
		if err != nil {
			// Not a date folder, skip
			continue
		}

		if folderDate.Before(cutoff) {
			folderPath := filepath.Join(dirPath, folderName)
			fmt.Printf("CleanupManager: Deleting old folder: %s\n", folderPath)
			if err := os.RemoveAll(folderPath); err == nil {
				deletedCount++
			} else {
				fmt.Printf("CleanupManager: Failed to delete folder: %s: %v\n", folderPath, err)
			}
		}
	}

	fmt.Printf("CleanupManager: Deletion finished. Successfully deleted %d folders older than %d days.\n", deletedCount, days)
	return deletedCount
}
