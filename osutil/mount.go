// -*- Mode: Go; indent-tabs-mode: t -*-

/*
 * Copyright (C) 2014-2016 Canonical Ltd
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License version 3 as
 * published by the Free Software Foundation.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 *
 */

package osutil

import (
	"fmt"
	"path/filepath"
	"strings"
	"syscall"
)

// IsMounted checks if a given directory is a mount point.
func IsMounted(baseDir string) (bool, error) {
	entries, err := LoadMountInfo(procSelfMountInfo())
	if err != nil {
		return false, err
	}
	for _, entry := range entries {
		if baseDir == entry.MountDir {
			return true, nil
		}
	}
	return false, nil
}

// not available through syscall
const (
	umountNoFollow = 8
)

// ConservativeMount performs a mount and performs additional checks.
//
// The following things are not allowed by conservative mount:
//
// 1) the mount point must be an absolute path
// 2) the bind-mount source must be an absolute path
// 3) the user must not have write permissions to the parent directory of the mount point.
// 4) mounting over an existing mount point.
// 5) mounting through a symlink (in final path component)
// 6) bind mounting from a symlink (in final path component)
func ConservativeMount(entry *MountEntry) error {
	flags, unparsed := MountOptsToCommonFlags(entry.Options)

	// Enforce rule 1)
	if !filepath.IsAbs(entry.Dir) {
		return fmt.Errorf("cannot use relative mount point %q", entry.Dir)
	}

	// Enforce rule 2)
	if flags&syscall.MS_BIND != 0 && !filepath.IsAbs(entry.Name) {
		return fmt.Errorf("cannot use relative bind-mount source %q", entry.Name)
	}

	// Enforce rule 3)
	err := descendFromRoot(entry.Dir, checkMountPoint)
	if err != nil {
		return err
	}

	// Enforce rule 4
	before, err := LoadMountInfo(procSelfMountInfo())
	if err != nil {
		return err
	}
	for _, mi := range before {
		if mi.MountDir == entry.Dir {
			return fmt.Errorf("cannot mount over existing mount entry %s", mi)
		}
	}

	// Perform the mount
	err = sysMount(entry.Name, entry.Dir, entry.Type, uintptr(flags), strings.Join(unparsed, ","))
	if err != nil {
		return err
	}

	// Enforce rule 5
	after, err := LoadMountInfo(procSelfMountInfo())
	if err != nil {
		return err
	}
	found := false
	for _, mi := range after {
		if mi.MountDir == entry.Dir {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("cannot ensure mount consistency: expected to find mounted %q", entry.Dir)
	}
	return nil
}

func checkMountPoint(fd, n, total int, path string) error {
	// fmt.Printf("check fd %d path: %q (%d of %d)\n", fd, path, n, total)
	if n != total - 1 {
		// Skip checks on all directories but the second-to-last.
		return nil
	}

	var stat syscall.Stat_t
	if err := sysFstat(fd, &stat); err != nil {
		return fmt.Errorf("cannot stat path segment: %s", err)
	}

	// TODO: if the user is not root then check if the user could overwrite
	// the last segment. This can be checked by looking at the second-to-last
	// segment and considering if it is writable.

	return nil
}

type checkFdPath func(fd, n, total int, path string) error

func descendFromRoot(path string, check checkFdPath) error {
	segments, err := splitIntoSegments(path)
	if err != nil {
		return fmt.Errorf("cannot descend from path %q: %s", path, err)
	}

	const openFlags = syscall.O_NOFOLLOW | syscall.O_CLOEXEC | syscall.O_DIRECTORY
	subPath := "/"
	fd, err := sysOpen(subPath, openFlags, 0)
	if err != nil {
		return fmt.Errorf("cannot open root directory: %v", err)
	}
	defer sysClose(fd)
	if err := check(fd, 0, len(segments), subPath); err != nil {
		return fmt.Errorf("cannot allow mount to traverse %q: %v", "/", err)
	}

	for i, segment := range segments {
		newFd, err := sysOpenat(fd, segment, openFlags, 0)
		if err != nil {
			return fmt.Errorf("cannot open path segment %q (got up to %q): %v", segment, subPath, err)
		}
		defer sysClose(newFd)
		fd = newFd
		if subPath == "/" {
			subPath = "/" + segment
		} else {
			subPath = subPath + "/" + segment
		}
		if err := check(fd, i+1, len(segments), subPath); err != nil {
			return fmt.Errorf("cannot allow mount to traverse %q: %v", subPath, err)
		}
	}
	return nil
}

func splitIntoSegments(name string) ([]string, error) {
	if name != filepath.Clean(name) {
		return nil, fmt.Errorf("cannot split unclean path %q", name)
	}
	segments := strings.FieldsFunc(filepath.Clean(name), func(c rune) bool { return c == '/' })
	return segments, nil
}
