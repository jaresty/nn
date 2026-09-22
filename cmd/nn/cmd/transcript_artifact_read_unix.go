//go:build darwin || linux

package cmd

import (
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/sys/unix"
)

func openTranscriptArtifact(root, path string) (artifactReadFile, error) {
	rootFD, err := openDirectoryChain(filepath.VolumeName(root)+string(filepath.Separator), strings.TrimPrefix(strings.TrimPrefix(root, filepath.VolumeName(root)), string(filepath.Separator)))
	if err != nil {
		return nil, err
	}
	defer unix.Close(rootFD)
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return nil, err
	}
	parts := strings.Split(rel, string(filepath.Separator))
	if len(parts) == 0 {
		return nil, os.ErrPermission
	}
	fd := rootFD
	owned := false
	for _, part := range parts[:len(parts)-1] {
		next, e := unix.Openat(fd, part, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
		if owned {
			unix.Close(fd)
		}
		if e != nil {
			return nil, e
		}
		fd = next
		owned = true
	}
	leaf, e := unix.Openat(fd, parts[len(parts)-1], unix.O_RDONLY|unix.O_NOFOLLOW|unix.O_NONBLOCK|unix.O_CLOEXEC, 0)
	if owned {
		unix.Close(fd)
	}
	if e != nil {
		return nil, e
	}
	return os.NewFile(uintptr(leaf), path), nil
}

func openDirectoryChain(base, rest string) (int, error) {
	fd, err := unix.Open(base, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
	if err != nil {
		return -1, err
	}
	for _, part := range strings.Split(rest, string(filepath.Separator)) {
		if part == "" {
			continue
		}
		next, e := unix.Openat(fd, part, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
		unix.Close(fd)
		if e != nil {
			return -1, e
		}
		fd = next
	}
	return fd, nil
}
