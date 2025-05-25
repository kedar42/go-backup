package main

import (
	"encoding/json"
	"log"
	"os"
	"strings"
	"time"
)

type Storage struct{}

func NewStorage() *Storage {
	return &Storage{}
}

func (s *Storage) WriteTimestamp(filePath string, timestamp time.Time) error {
	data := timestamp.Format(time.RFC3339)
	if err := os.WriteFile(filePath, []byte(data), 0644); err != nil {
		log.Printf("Error writing timestamp to file: %v", err)
		return err
	}
	log.Printf("Timestamp saved to file: %s", filePath)
	return nil
}

func (s *Storage) ReadTimestamp(filePath string) (time.Time, error) {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		log.Printf("Timestamp file %s does not exist", filePath)
		return time.Time{}, err
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		log.Printf("Error reading timestamp file: %v", err)
		return time.Time{}, err
	}

	timestamp, err := time.Parse(time.RFC3339, strings.TrimSpace(string(data)))
	if err != nil {
		log.Printf("Error parsing timestamp: %v", err)
		return time.Time{}, err
	}

	return timestamp, nil
}

func (s *Storage) WriteFiles(files []string, filePath string) error {
	data, err := json.MarshalIndent(files, "", "  ")
	if err != nil {
		log.Printf("Error marshaling files to JSON: %v", err)
		return err
	}

	if err := os.WriteFile(filePath, data, 0644); err != nil {
		log.Printf("Error writing files to storage: %v", err)
		return err
	}

	return nil
}

func (s *Storage) ReadFiles(filePath string) ([]string, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var files []string
	if err := json.Unmarshal(data, &files); err != nil {
		return nil, err
	}

	return files, nil
}
