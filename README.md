# go-backup

This simple backup tool is designed to periodically copy files from source. It uses `rclone` to perform the actual copying, the main value this program adds is automatic scheduling and checking if the upstream needs to be copied.

It was mainly developed by me to use in my homelab and I wanted to keep as much of that in my docker-compose as possible, so it is designed to be run in a Docker container. Not sure if anyone else will find it useful, but if you do, feel free to use it.

## TODO
- [x] Fix bug where for some reason changes were detected but nothing was copied
- [x] Ensure that on force backup, --min-age is not used
- [x] Fix output of `rsync copy` command
- [x] Add log info how many files were copied and how long it took
- [ ] Add signing of Docker images
