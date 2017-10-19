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

package main_test

import (
	"os"
	"syscall"

	. "gopkg.in/check.v1"

	update "github.com/snapcore/snapd/cmd/snap-update-ns"
	"github.com/snapcore/snapd/interfaces/mount"
	"github.com/snapcore/snapd/testutil"
)

type utilsSuite struct {
	testutil.BaseTest
	sys *update.SyscallRecorder
}

var _ = Suite(&utilsSuite{})

func (s *utilsSuite) SetUpTest(c *C) {
	s.BaseTest.SetUpTest(c)
	s.sys = &update.SyscallRecorder{}
	s.BaseTest.AddCleanup(update.MockSystemCalls(s.sys))
}

func (s *utilsSuite) TearDownTest(c *C) {
	s.sys.CheckForStrayDescriptors(c)
	s.BaseTest.TearDownTest(c)
}

// Ensure that we refuse to create a directory with an relative path.
func (s *utilsSuite) TestSecureMkdirAllRelative(c *C) {
	err := update.SecureMkdirAll("rel/path", 0755, 123, 456)
	c.Assert(err, ErrorMatches, `cannot create directory with relative path: "rel/path"`)
	c.Assert(s.sys.Calls(), HasLen, 0)
}

// Ensure that we can create a directory with an absolute path.
func (s *utilsSuite) TestSecureMkdirAllAbsolute(c *C) {
	c.Assert(update.SecureMkdirAll("/abs/path", 0755, 123, 456), IsNil)
	c.Assert(s.sys.Calls(), DeepEquals, []string{
		`open "/" O_NOFOLLOW|O_CLOEXEC|O_DIRECTORY 0`,
		`mkdirat 3 "abs" 0755`,
		`openat 3 "abs" O_NOFOLLOW|O_CLOEXEC|O_DIRECTORY 0`,
		`fchown 4 123 456`,
		`mkdirat 4 "path" 0755`,
		`openat 4 "path" O_NOFOLLOW|O_CLOEXEC|O_DIRECTORY 0`,
		`fchown 5 123 456`,
		`close 5`,
		`close 4`,
		`close 3`,
	})
}

// Ensure that we can detect read only filesystems.
func (s *utilsSuite) TestSecureMkdirAllROFS(c *C) {
	s.sys.InsertFault(`mkdirat 4 "path" 0755`, syscall.EROFS)
	err := update.SecureMkdirAll("/rofs/path", 0755, 123, 456)
	c.Assert(err, ErrorMatches, `cannot operate on read-only filesystem at /rofs`)
	c.Assert(err.(*update.ReadOnlyFsError).Path, Equals, "/rofs")
	c.Assert(s.sys.Calls(), DeepEquals, []string{
		`open "/" O_NOFOLLOW|O_CLOEXEC|O_DIRECTORY 0`,
		`mkdirat 3 "rofs" 0755`,
		`openat 3 "rofs" O_NOFOLLOW|O_CLOEXEC|O_DIRECTORY 0`,
		`fchown 4 123 456`,
		`mkdirat 4 "path" 0755`,
		`close 4`,
		`close 3`,
	})
}

// Ensure that we don't chown existing directories.
func (s *utilsSuite) TestSecureMkdirAllExistingDirsDontChown(c *C) {
	s.sys.InsertFault(`mkdirat 3 "abs" 0755`, syscall.EEXIST)
	s.sys.InsertFault(`mkdirat 4 "path" 0755`, syscall.EEXIST)
	err := update.SecureMkdirAll("/abs/path", 0755, 123, 456)
	c.Assert(err, IsNil)
	c.Assert(s.sys.Calls(), DeepEquals, []string{
		`open "/" O_NOFOLLOW|O_CLOEXEC|O_DIRECTORY 0`,
		`mkdirat 3 "abs" 0755`,
		`openat 3 "abs" O_NOFOLLOW|O_CLOEXEC|O_DIRECTORY 0`,
		`mkdirat 4 "path" 0755`,
		`openat 4 "path" O_NOFOLLOW|O_CLOEXEC|O_DIRECTORY 0`,
		`close 5`,
		`close 4`,
		`close 3`,
	})
}

// Ensure that we we close everything when mkdir fails.
func (s *utilsSuite) TestSecureMkdirAllCloseOnError(c *C) {
	s.sys.InsertFault(`mkdirat 3 "abs" 0755`, errTesting)
	err := update.SecureMkdirAll("/abs", 0755, 123, 456)
	c.Assert(err, ErrorMatches, `cannot mkdir path segment "abs": testing`)
	c.Assert(s.sys.Calls(), DeepEquals, []string{
		`open "/" O_NOFOLLOW|O_CLOEXEC|O_DIRECTORY 0`,
		`mkdirat 3 "abs" 0755`,
		`close 3`,
	})
}

// Explore how mounting overlay looks like technically
func (s *utilsSuite) TestMountOverlayAt(c *C) {
	change, err := update.MountOverlayAt("/abs/path")
	c.Assert(err, IsNil)
	c.Assert(change, DeepEquals, &update.Change{
		Action: update.Mount,
		Entry: mount.Entry{
			Name: "none",
			Dir:  "/abs/path",
			Type: "overlay",
			Options: []string{
				"lowerdir=/abs/path",
				"upperdir=/tmp/.snap.overlays/abs/path",
				"workdir=/tmp/.snap.workdirs/abs/path",
			},
		},
	})
	c.Assert(s.sys.Calls(), DeepEquals, []string{
		// Create "/tmp/.snap.overlays/abs/path".
		`open "/" O_NOFOLLOW|O_CLOEXEC|O_DIRECTORY 0`,
		`mkdirat 3 "tmp" 0755`,
		`openat 3 "tmp" O_NOFOLLOW|O_CLOEXEC|O_DIRECTORY 0`,
		`fchown 4 0 0`,
		`mkdirat 4 ".snap.overlays" 0755`,
		`openat 4 ".snap.overlays" O_NOFOLLOW|O_CLOEXEC|O_DIRECTORY 0`,
		`fchown 5 0 0`,
		`mkdirat 5 "abs" 0755`,
		`openat 5 "abs" O_NOFOLLOW|O_CLOEXEC|O_DIRECTORY 0`,
		`fchown 6 0 0`,
		`mkdirat 6 "path" 0755`,
		`openat 6 "path" O_NOFOLLOW|O_CLOEXEC|O_DIRECTORY 0`,
		`fchown 7 0 0`,
		`close 7`,
		`close 6`,
		`close 5`,
		`close 4`,
		`close 3`,
		// Create "/tmp/.snap.workdirs/abs/path".
		`open "/" O_NOFOLLOW|O_CLOEXEC|O_DIRECTORY 0`,
		`mkdirat 3 "tmp" 0755`,
		`openat 3 "tmp" O_NOFOLLOW|O_CLOEXEC|O_DIRECTORY 0`,
		`fchown 4 0 0`,
		`mkdirat 4 ".snap.workdirs" 0755`,
		`openat 4 ".snap.workdirs" O_NOFOLLOW|O_CLOEXEC|O_DIRECTORY 0`,
		`fchown 5 0 0`,
		`mkdirat 5 "abs" 0755`,
		`openat 5 "abs" O_NOFOLLOW|O_CLOEXEC|O_DIRECTORY 0`,
		`fchown 6 0 0`,
		`mkdirat 6 "path" 0755`,
		`openat 6 "path" O_NOFOLLOW|O_CLOEXEC|O_DIRECTORY 0`,
		`fchown 7 0 0`,
		`close 7`,
		`close 6`,
		`close 5`,
		`close 4`,
		`close 3`,
		// Mount overlay at /abs/path with appropriate {lower,upper,work}dirs.
		`mount "none" "/abs/path" "overlay" 0 "lowerdir=/abs/path,upperdir=/tmp/.snap.overlays/abs/path,workdir=/tmp/.snap.workdirs/abs/path"`,
	})
}

// Ensure that we cannot put overlays over any of the restricted locations.
func (s *utilsSuite) TestMountOverlayAtRestrictedLocation(c *C) {
	for _, dir := range []string{"/tmp", "/tmp/", "/tmp/stuff"} {
		change, err := update.MountOverlayAt(dir)
		c.Assert(err, ErrorMatches, `refusing to create overlay at ".*"`, Commentf("dir: %q", dir))
		c.Assert(change, IsNil)
	}
}

// Ensure that errors are handled.

func (s *utilsSuite) TestMountOverlayAtErrors1(c *C) {
	s.sys.InsertFault(`mkdirat 4 ".snap.overlays" 0755`, syscall.EPERM)
	_, err := update.MountOverlayAt("/abs/path")
	c.Assert(err, ErrorMatches, `cannot mkdir path segment ".snap.overlays": operation not permitted`)
	c.Assert(s.sys.Calls(), Not(testutil.Contains),
		`mount "none" "/abs/path" "overlay" 0 "lowerdir=/abs/path,upperdir=/tmp/.snap.overlays/abs/path,workdir=/tmp/.snap.workdirs/abs/path"`)
}

func (s *utilsSuite) TestMountOverlayAtErrors2(c *C) {
	s.sys.InsertFault(`mkdirat 4 ".snap.workdirs" 0755`, syscall.EPERM)
	_, err := update.MountOverlayAt("/abs/path")
	c.Assert(err, ErrorMatches, `cannot mkdir path segment ".snap.workdirs": operation not permitted`)
	c.Assert(s.sys.Calls(), Not(testutil.Contains),
		`mount "none" "/abs/path" "overlay" 0 "lowerdir=/abs/path,upperdir=/tmp/.snap.overlays/abs/path,workdir=/tmp/.snap.workdirs/abs/path"`)
}

func (s *utilsSuite) TestMountOverlayAtErrors3(c *C) {
	s.sys.InsertFault(`mount "none" "/abs/path" "overlay" 0 "lowerdir=/abs/path,upperdir=/tmp/.snap.overlays/abs/path,workdir=/tmp/.snap.workdirs/abs/path"`, syscall.EPERM)
	_, err := update.MountOverlayAt("/abs/path")
	c.Assert(err, ErrorMatches, `cannot mount overlay at "/abs/path": operation not permitted`)
}

func (s *utilsSuite) TestEnsureMountPointSuperExistingDir(c *C) {
	s.sys.InsertLstatResult(`lstat "/abs/path"`, update.FileInfoDir)
	change, err := update.EnsureMountPointMaybeUsingOverlay("/abs/path", 0755, 0, 0)
	c.Assert(err, IsNil)
	c.Assert(change, IsNil)
}

func (s *utilsSuite) TestEnsureMountPointSuperExistingFile(c *C) {
	s.sys.InsertLstatResult(`lstat "/abs/path"`, update.FileInfoFile)
	change, err := update.EnsureMountPointMaybeUsingOverlay("/abs/path", 0755, 0, 0)
	c.Assert(err, ErrorMatches, `cannot use "/abs/path" for mounting, not a directory`)
	c.Assert(change, IsNil)
}

func (s *utilsSuite) TestEnsureMountPointSuperExistingSymlink(c *C) {
	s.sys.InsertLstatResult(`lstat "/abs/path"`, update.FileInfoSymlink)
	change, err := update.EnsureMountPointMaybeUsingOverlay("/abs/path", 0755, 0, 0)
	c.Assert(err, ErrorMatches, `cannot use "/abs/path" for mounting, not a directory`)
	c.Assert(change, IsNil)
}

func (s *utilsSuite) TestEnsureMountPointSuperPermissionDenied(c *C) {
	s.sys.InsertFault(`lstat "/abs/path"`, syscall.EPERM)
	change, err := update.EnsureMountPointMaybeUsingOverlay("/abs/path", 0755, 0, 0)
	c.Assert(err, ErrorMatches, `cannot inspect "/abs/path": operation not permitted`)
	c.Assert(change, IsNil)
}

func (s *utilsSuite) TestEnsureMountPointSuperROFS(c *C) {
	var n, m int
	s.sys.InsertFaultFunc(`lstat "/abs/path"`, func() error {
		n += 1
		if n == 1 {
			return os.ErrNotExist
		}
		return nil
	})
	s.sys.InsertLstatResult(`lstat "/abs/path"`, update.FileInfoDir)
	s.sys.InsertFaultFunc(`mkdirat 4 "path" 0755`, func() error {
		m += 1
		if m == 1 {
			return syscall.EROFS
		}
		return nil
	})
	change, err := update.EnsureMountPointMaybeUsingOverlay("/abs/path", 0755, 0, 0)
	c.Assert(err, IsNil)
	c.Assert(change, DeepEquals, &update.Change{
		Action: update.Mount,
		Entry: mount.Entry{
			Name: "none",
			Dir:  "/abs",
			Type: "overlay",
			Options: []string{
				"lowerdir=/abs",
				"upperdir=/tmp/.snap.overlays/abs",
				"workdir=/tmp/.snap.workdirs/abs",
			},
		},
	})
	c.Assert(s.sys.Calls(), testutil.Contains,
		`mount "none" "/abs" "overlay" 0 "lowerdir=/abs,upperdir=/tmp/.snap.overlays/abs,workdir=/tmp/.snap.workdirs/abs"`)
}
