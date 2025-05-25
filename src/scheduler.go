package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

type Scheduler struct {
	rclone      *RcloneClient
	fileTracker *FileTracker
	storage     *Storage
}

func NewScheduler() *Scheduler {
	storage := NewStorage()
	return &Scheduler{
		rclone:      NewRcloneClient(),
		fileTracker: NewFileTracker(storage),
		storage:     storage,
	}
}

func (s *Scheduler) Run(cfg Config) error {
	log.Printf("Starting backup scheduler with configuration:")
	s.logConfig(cfg)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	ticker := time.NewTicker(cfg.Interval)
	defer ticker.Stop()

	s.runBackupCheck(cfg)

	for {
		select {
		case <-ctx.Done():
			log.Println("Context cancelled, shutting down...")
			return nil
		case <-sigChan:
			log.Println("Received interrupt signal, shutting down gracefully...")
			cancel()
			return nil
		case <-ticker.C:
			s.runBackupCheck(cfg)
		}
	}
}

func (s *Scheduler) runBackupCheck(cfg Config) {
	log.Println("Starting backup check...")

	files, err := s.rclone.ListFiles(cfg.Source, cfg.RcloneConfigPath)
	if err != nil {
		log.Printf("Failed to list files: %v", err)
		return
	}

	log.Printf("Found %d files in source directory", len(files))

	forceBackup := s.fileTracker.ShouldForceBackup(cfg.LastForceFile, cfg.ForceInterval)
	hasChanges := s.fileTracker.HasChanges(files, cfg.PreviousFilesJSON)

	if hasChanges || forceBackup {
		s.performBackup(cfg, files, hasChanges, forceBackup)
	} else {
		log.Println("No changes detected and no forced backup needed, skipping backup")
	}
}

func (s *Scheduler) performBackup(cfg Config, files []string, hasChanges, forceBackup bool) {
	reason := s.getBackupReason(hasChanges, forceBackup)
	log.Printf("Performing backup: %s", reason)

	if err := s.fileTracker.SaveCurrentFiles(files, cfg.PreviousFilesJSON); err != nil {
		log.Printf("Failed to save file list: %v", err)
		return
	}

	lastRunTime := s.fileTracker.GetLastRunTime(cfg.TimestampFile)

	if err := s.rclone.CopyFiles(cfg.Source, cfg.Destination, lastRunTime, cfg.OverlapBuffer, cfg.RcloneConfigPath); err != nil {
		log.Printf("Backup failed: %v", err)
		return
	}

	if err := s.fileTracker.SaveRunTimestamp(cfg.TimestampFile); err != nil {
		log.Printf("Failed to save run timestamp: %v", err)
	}

	if forceBackup {
		if err := s.fileTracker.SaveRunTimestamp(cfg.LastForceFile); err != nil {
			log.Printf("Failed to save force timestamp: %v", err)
		}
	}

	log.Println("Backup completed successfully")
}

func (s *Scheduler) getBackupReason(hasChanges, forceBackup bool) string {
	switch {
	case hasChanges && forceBackup:
		return "changes detected + forced backup due"
	case forceBackup:
		return "forced backup (no changes detected)"
	default:
		return "changes detected"
	}
}

func (s *Scheduler) logConfig(cfg Config) {
	log.Printf("Source: %s", cfg.Source)
	log.Printf("Destination: %s", cfg.Destination)
	log.Printf("Check interval: %v", cfg.Interval)
	log.Printf("Force interval: %v", cfg.ForceInterval)
	log.Printf("Overlap buffer: %v", cfg.OverlapBuffer)
	if cfg.RcloneConfigPath != "" {
		log.Printf("Rclone config: %s", cfg.RcloneConfigPath)
	} else {
		log.Printf("Rclone config: using default location")
	}
}
