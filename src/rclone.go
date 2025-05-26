package main

import (
	"log"
	"os/exec"
	"strings"
	"time"
)

type RcloneClient struct{}

func NewRcloneClient() *RcloneClient {
	return &RcloneClient{}
}

func (r *RcloneClient) ListFiles(source, configPath string) ([]string, error) {
	args := []string{"lsf", source, "-R", "--files-only"}
	if configPath != "" {
		args = append([]string{"--config", configPath}, args...)
	}

	cmd := exec.Command("rclone", args...)
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	files := strings.Split(strings.TrimSpace(string(output)), "\n")
	return r.filterEmptyFiles(files), nil
}

func (r *RcloneClient) CopyFiles(source, destination, lastRun string, overlapBuffer time.Duration, configPath string, forced bool) error {
	args := []string{"copy", source, destination, "--stats-one-line", "--log-file", "/tmp/rclone.log", "-v"}

	if configPath != "" {
		args = append([]string{"--config", configPath}, args...)
	}

	if lastRun != "" && !forced {
		if adjustedTime, err := r.calculateAdjustedTime(lastRun, overlapBuffer); err == nil {
			args = append(args, "--max-age", adjustedTime)
			log.Printf("Using --max-age with overlap: original=%s, adjusted=%s (buffer: %v)",
				lastRun, adjustedTime, overlapBuffer)
		} else {
			log.Printf("Error parsing last run time, proceeding without --max-age: %v", err)
		}
	} else {
		log.Printf("Forced copy or no previous run timestamp, copying all files")
	}

	startTime := time.Now()

	log.Printf("Executing: rclone %s", strings.Join(args, " "))
	cmd := exec.Command("rclone", args...)

	if err := cmd.Start(); err != nil {
		log.Printf("Failed to start rclone: %v", err)
		return err
	}

	if err := cmd.Wait(); err != nil {
		log.Printf("Rclone exited with error: %v", err)
		return err
	}

	duration := time.Since(startTime)
	log.Printf("Files copied successfully in %s", duration)

	return nil
}

func (r *RcloneClient) filterEmptyFiles(files []string) []string {
	var validFiles []string
	for _, file := range files {
		if strings.TrimSpace(file) != "" {
			validFiles = append(validFiles, file)
		}
	}
	return validFiles
}

func (r *RcloneClient) calculateAdjustedTime(lastRun string, overlapBuffer time.Duration) (string, error) {
	lastRunTime, err := time.Parse(time.RFC3339, lastRun)
	if err != nil {
		return "", err
	}
	adjustedTime := lastRunTime.Add(-overlapBuffer)
	return adjustedTime.Format(time.RFC3339), nil
}
