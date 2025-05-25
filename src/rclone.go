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

func (r *RcloneClient) CopyFiles(source, destination, lastRun string, overlapBuffer time.Duration, configPath string) error {
	args := []string{"copy", source, destination, "--stats-one-line-date"}

	if configPath != "" {
		args = append([]string{"--config", configPath}, args...)
	}

	if lastRun != "" {
		if adjustedTime, err := r.calculateAdjustedTime(lastRun, overlapBuffer); err == nil {
			args = append(args, "--min-age", adjustedTime)
			log.Printf("Using --min-age with overlap: original=%s, adjusted=%s (buffer: %v)",
				lastRun, adjustedTime, overlapBuffer)
		} else {
			log.Printf("Error parsing last run time, proceeding without --min-age: %v", err)
		}
	} else {
		log.Println("No previous run timestamp, copying all files")
	}

	log.Printf("Executing: rclone %s", strings.Join(args, " "))
	cmd := exec.Command("rclone", args...)
	output, err := cmd.CombinedOutput()

	if err != nil {
		log.Printf("Error copying files: %v", err)
		log.Printf("Rclone output: %s", string(output))
		return err
	}

	if len(output) > 0 {
		log.Printf("Copy operation completed:\n%s", string(output))
	} else {
		log.Println("Copy operation completed successfully (no output)")
	}

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
