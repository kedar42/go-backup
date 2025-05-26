# go-backup

This is my personal backup script for my home server. It is written in Go and uses the `rsync` command to perform backups.

## TODO
- [x] Fix bug where for some reason changes were detected but nothing was copied
- [x] Ensure that on force backup, --min-age is not used
- [ ] Fix output of `rsync copy` command
- [ ] Add log info how many files were copied and how long it took
- [ ] Add signing of Docker images
