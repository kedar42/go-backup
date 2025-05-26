package main

import (
	"log"
	"time"
)

type FileTracker struct {
	storage *Storage
}

func NewFileTracker(storage *Storage) *FileTracker {
	return &FileTracker{
		storage: storage,
	}
}

func (ft *FileTracker) HasChanges(currentFiles []string, previousFilesPath string) bool {
	previousFiles, err := ft.storage.ReadFiles(previousFilesPath)
	if err != nil {
		log.Printf("Error reading previous files: %v", err)
		return true
	}

	if len(previousFiles) == 0 {
		log.Println("No previous files to compare with")
		return len(currentFiles) > 0
	}

	previousFileMap := make(map[string]bool)
	for _, file := range previousFiles {
		previousFileMap[file] = true
	}

	for _, currentFile := range currentFiles {
		if !previousFileMap[currentFile] {
			log.Printf("New file detected: %s", currentFile)
			return true
		}
	}

	log.Println("No new files detected, backup not needed")
	return false
}

func (ft *FileTracker) SaveCurrentFiles(files []string, filePath string) error {
	if err := ft.storage.WriteFiles(files, filePath); err != nil {
		return err
	}
	log.Printf("Current files saved to: %s", filePath)
	return nil
}

func (ft *FileTracker) ShouldForceBackup(lastForceFile string, forceInterval time.Duration) bool {
	lastForce, err := ft.storage.ReadTimestamp(lastForceFile)
	if err != nil {
		log.Println("No previous forced backup found, will force backup")
		return true
	}

	timeSinceForce := time.Since(lastForce)
	if timeSinceForce >= forceInterval {
		log.Printf("Time since last forced backup: %v (>= %v), forcing backup", timeSinceForce, forceInterval)
		return true
	}

	log.Printf("Time since last forced backup: %v (< %v), no force needed", timeSinceForce, forceInterval)
	return false
}

func (ft *FileTracker) GetLastRunTime(timestampFile string) string {
	lastRunTime, err := ft.storage.ReadTimestamp(timestampFile)
	if err != nil {
		return ""
	}
	return lastRunTime.Format(time.RFC3339)
}

func (ft *FileTracker) SaveRunTimestamp(timestampFile string) error {
	return ft.storage.WriteTimestamp(timestampFile, time.Now())
}
