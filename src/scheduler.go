package main

import (
	"bufio"
	"context"
	"io"
	"log"
	"os"
	"os/signal"
	"regexp"
	"strings"
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

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go watchForFirstETA(ctx, "/tmp/rclone.log")

	if err := s.rclone.CopyFiles(cfg.Source, cfg.Destination, lastRunTime, cfg.OverlapBuffer, cfg.RcloneConfigPath, forceBackup); err != nil {
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

	if totalSize, copiedItems, err := parseRcloneLog("/tmp/rclone.log"); err == nil {
		log.Printf("Total size: %s, Copied items: %d", totalSize, copiedItems)
	} else {
		log.Printf("Failed to parse rclone log: %v", err)
	}

	err := os.Remove("/tmp/rclone.log")
	if err != nil {
		log.Printf("Failed to remove rclone log file: %v", err)
	}
}

func watchForFirstETA(ctx context.Context, logFilePath string) {
	file, err := os.Open(logFilePath)
	if err != nil {
		log.Printf("Failed to open rclone log for monitoring: %v", err)
		return
	}
	defer file.Close()

	file.Seek(0, io.SeekEnd)
	reader := bufio.NewReader(file)
	etaRegex := regexp.MustCompile(`ETA\s+([0-9]+[hms]+(?:[0-9]+[ms]+)*)`)

	for {
		select {
		case <-ctx.Done():
			return
		default:
			line, err := reader.ReadString('\n')
			if err != nil {
				if err == io.EOF {
					time.Sleep(200 * time.Millisecond)
					continue
				}
				log.Printf("Error reading log file: %v", err)
				return
			}

			if matches := etaRegex.FindStringSubmatch(line); matches != nil {
				log.Printf("First ETA found: %s", matches[1])
				return
			}
		}
	}
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

func parseRcloneLog(filename string) (string, int, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return "", 0, err
	}

	progressPattern := regexp.MustCompile(`(\d+\.\d+\s+[KMGT]?i?B)\s+/\s+(\d+\.\d+\s+[KMGT]?i?B)`)
	copiedPattern := regexp.MustCompile(`Multi-thread Copied \(new\)`)

	var totalSize string
	copiedItems := 0

	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		if matches := progressPattern.FindStringSubmatch(line); matches != nil {
			totalSize = matches[2]
		}

		if copiedPattern.MatchString(line) {
			copiedItems++
		}
	}
	return totalSize, copiedItems, nil
}
