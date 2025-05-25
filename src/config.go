package main

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
)

type Config struct {
	Interval          time.Duration
	Source            string
	Destination       string
	TimestampFile     string
	PreviousFilesJSON string
	ForceInterval     time.Duration
	LastForceFile     string
	OverlapBuffer     time.Duration
	RcloneConfigPath  string
}

const (
	defaultInterval          = 30 * time.Minute
	defaultTimestampFile     = "last_run.txt"
	defaultPreviousFilesJSON = "previous_files.json"
	defaultForceInterval     = 24 * time.Hour
	defaultLastForceFile     = "last_force.txt"
	defaultOverlapBuffer     = 5 * time.Minute
)

func NewRootCommand() *cobra.Command {
	var config Config

	rootCmd := &cobra.Command{
		Use:   "backup-scheduler",
		Short: "Automated backup scheduler using rclone",
		Long: `A smart backup scheduler that monitors changes and performs incremental backups using rclone.
It supports both local and remote destinations, with configurable intervals and overlap buffers.`,
		Example: `  # Local to local backup
  backup-scheduler --source /home/user/docs --dest /backup/docs
  # Remote to local backup with custom config
  backup-scheduler --source gdrive:Documents --dest /local/backup --rclone-config /path/to/rclone.conf
  # Custom intervals and overlap
  backup-scheduler --source /data --dest s3:mybucket/backup --interval 15m --overlap-buffer 10m`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validateConfig(config); err != nil {
				return err
			}
			scheduler := NewScheduler()
			return scheduler.Run(config)
		},
	}

	setupFlags(rootCmd, &config)
	return rootCmd
}

func setupFlags(cmd *cobra.Command, config *Config) {
	cmd.Flags().DurationVar(&config.Interval, "interval", defaultInterval,
		"Backup check interval (e.g., 5m, 1h, 30s)")
	cmd.Flags().StringVar(&config.Source, "source", "",
		"Source directory/remote to backup (REQUIRED)")
	cmd.Flags().StringVar(&config.Destination, "dest", "",
		"Destination directory/remote for backup (REQUIRED)")
	cmd.Flags().StringVar(&config.TimestampFile, "timestamp-file", defaultTimestampFile,
		"File to store last run timestamp")
	cmd.Flags().StringVar(&config.PreviousFilesJSON, "files-cache", defaultPreviousFilesJSON,
		"File to cache previous file list")
	cmd.Flags().DurationVar(&config.ForceInterval, "force-interval", defaultForceInterval,
		"Interval for forced backup regardless of changes")
	cmd.Flags().StringVar(&config.LastForceFile, "force-file", defaultLastForceFile,
		"File to store last forced backup timestamp")
	cmd.Flags().DurationVar(&config.OverlapBuffer, "overlap-buffer", defaultOverlapBuffer,
		"Time buffer to overlap with previous run to avoid missing files")
	cmd.Flags().StringVar(&config.RcloneConfigPath, "rclone-config", "",
		"Path to rclone config file (optional, uses rclone default if not specified)")

	cmd.MarkFlagRequired("source")
	cmd.MarkFlagRequired("dest")
}

func validateConfig(config Config) error {
	if config.RcloneConfigPath != "" {
		if _, err := os.Stat(config.RcloneConfigPath); os.IsNotExist(err) {
			return fmt.Errorf("rclone config file does not exist: %s", config.RcloneConfigPath)
		}
	}
	return nil
}
