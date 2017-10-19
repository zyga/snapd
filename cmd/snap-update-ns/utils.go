// -*- Mode: Go; indent-tabs-mode: t -*-

/*
 * Copyright (C) 2017 Canonical Ltd
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

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/snapcore/snapd/interfaces/mount"
)

// not available through syscall
const (
	UMOUNT_NOFOLLOW = 8
)

// For mocking everything during testing.
var (
	osLstat = os.Lstat

	sysClose   = syscall.Close
	sysMkdirat = syscall.Mkdirat
	sysMount   = syscall.Mount
	sysOpen    = syscall.Open
	sysOpenat  = syscall.Openat
	sysUnmount = syscall.Unmount
	sysFchown  = syscall.Fchown
)

// ReadOnlyFsError is an error encapsulating encountered EROFS.
type ReadOnlyFsError struct {
	Path string
}

func (e *ReadOnlyFsError) Error() string {
	return fmt.Sprintf("cannot operate on read-only filesystem at %s", e.Path)
}

// SecureMkdirAll is the secure variant of os.MkdirAll.
//
// Unlike the regular version this implementation does not follow any symbolic
// links. At all times the new directory segment is created using mkdirat(2)
// while holding an open file descriptor to the parent directory.
//
// The only handled error is mkdirat(2) that fails with EEXIST. All other
// errors are fatal but there is no attempt to undo anything that was created.
//
// The uid and gid are used for the fchown(2) system call which is performed
// after each segment is created and opened. The special value -1 may be used
// to request that ownership is not changed.
func secureMkdirAll(name string, perm os.FileMode, uid, gid int) error {
	// Declare var and don't assign-declare below to ensure we don't swallow
	// any errors by mistake.
	var err error
	var fd int

	const openFlags = syscall.O_NOFOLLOW | syscall.O_CLOEXEC | syscall.O_DIRECTORY

	// Only support absolute paths to avoid bugs in snap-confine when
	// called from anywhere.
	if !filepath.IsAbs(name) {
		return fmt.Errorf("cannot create directory with relative path: %q", name)
	}
	// Open the root directory and start there.
	fd, err = sysOpen("/", openFlags, 0)
	if err != nil {
		return fmt.Errorf("cannot open root directory: %v", err)
	}
	defer sysClose(fd)

	// Split the path by entries and create each element using mkdirat() using
	// the parent directory as reference. Each time we open the newly created
	// segment using the O_NOFOLLOW and O_DIRECTORY flag so that symlink
	// attacks are impossible to carry out.
	segments := strings.FieldsFunc(filepath.Clean(name), func(c rune) bool { return c == '/' })
	for i, segment := range segments {
		made := true
		if err = sysMkdirat(fd, segment, uint32(perm)); err != nil {
			switch err {
			case syscall.EEXIST:
				made = false
			case syscall.EROFS:
				// Treat EROFS specially: this is a hint that we have to poke a
				// hole using overlayfs. The path below is the location where
				// we need to poke the hole.
				p := "/" + strings.Join(segments[:i], "/")
				return &ReadOnlyFsError{Path: p}
			default:
				return fmt.Errorf("cannot mkdir path segment %q: %v", segment, err)
			}
		}
		fd, err = sysOpenat(fd, segment, openFlags, 0)
		if err != nil {
			return fmt.Errorf("cannot open path segment %q (got up to %q): %v", segment,
				"/"+strings.Join(segments[:i], "/"), err)
		}
		defer sysClose(fd)
		if made {
			// Chown each segment that we made.
			if err := sysFchown(fd, uid, gid); err != nil {
				return fmt.Errorf("cannot chown path segment %q to %d.%d: %v", segment, uid, gid, err)
			}
		}

	}
	return nil
}

func mountOverlayAt(dir string) (*Change, error) {
	// Overlay uses three directories: lower, upper and work.
	// Lower is our read-only substrate Upper is an ephemeral
	// sub-directory of /tmp (see below). Work is an empty
	// sibling of upper, needed by overlayfs to function.
	//
	// The lower directory cannot be in /tmp as we use (private) /tmp for the
	// overlay machinery itself.
	//
	// The upper directory is already on top of an existing host-based /tmp
	// directory, thanks to how snap-confine is arranigng the per-snap, private
	// /tmp directory.
	lowerDir := dir
	upperDir := filepath.Join("/tmp/.snap.overlays", dir)
	workDir := filepath.Join("/tmp/.snap.workdirs", dir)

	for _, blacklistDir := range []string{"/tmp"} {
		if strings.HasPrefix(lowerDir, blacklistDir+"/") || lowerDir == blacklistDir {
			return nil, fmt.Errorf("refusing to create overlay at %q", lowerDir)
		}
	}

	// Create upper and work directories.
	if err := secureMkdirAll(upperDir, 0755, 0, 0); err != nil {
		return nil, err
	}
	if err := secureMkdirAll(workDir, 0755, 0, 0); err != nil {
		return nil, err
	}

	// Create and perform and return a change describing the overlay mount.
	change := &Change{
		Action: Mount,
		Entry: mount.Entry{
			Name: "none",
			Dir:  dir,
			Type: "overlay",
			Options: []string{
				// Format the options, note the mount escape logic for paths.
				fmt.Sprintf("lowerdir=%s", mount.Escape(lowerDir)),
				fmt.Sprintf("upperdir=%s", mount.Escape(upperDir)),
				fmt.Sprintf("workdir=%s", mount.Escape(workDir)),
			},
		},
	}
	if err := change.lowLevelPerform(); err != nil {
		return nil, fmt.Errorf("cannot mount overlay at %q: %v", dir, err)
	}
	return change, nil
}

func ensureMountPoint(path string, mode os.FileMode, uid int, gid int) error {
	// If the mount point is not present then create a directory in its
	// place.  This is very naive, doesn't handle read-only file systems
	// but it is a good starting point for people working with things like
	// $SNAP_DATA/subdirectory.
	//
	// We use lstat to ensure that we don't follow the symlink in case one
	// was set up by the snap. Note that at the time this is run, all the
	// snap's processes are frozen.
	fi, err := osLstat(path)
	switch {
	case err != nil && os.IsNotExist(err):
		return secureMkdirAll(path, mode, uid, gid)
	case err != nil:
		return fmt.Errorf("cannot inspect %q: %v", path, err)
	case err == nil:
		// Ensure that mount point is a directory.
		if !fi.IsDir() {
			return fmt.Errorf("cannot use %q for mounting, not a directory", path)
		}
	}
	return nil
}

func ensureMountPointMaybeUsingOverlay(dir string, mode os.FileMode, uid int, gid int) (*Change, error) {
	var extraChange *Change
	err := ensureMountPoint(dir, mode, uid, gid)
	if err != nil {
		if err2, ok := err.(*ReadOnlyFsError); ok {
			extraChange, err = mountOverlayAt(err2.Path)
		}
		if err == nil {
			err = ensureMountPoint(dir, mode, uid, gid)
		}
	}
	return extraChange, err
}
