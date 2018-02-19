// -*- Mode: Go; indent-tabs-mode: t -*-

/*
 * Copyright (C) 2016 Canonical Ltd
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

package osutil_test

import (
	"bytes"
	"syscall"

	. "gopkg.in/check.v1"

	"github.com/snapcore/snapd/logger"
	"github.com/snapcore/snapd/osutil"
	"github.com/snapcore/snapd/testutil"
)

type mountSuite struct {
	testutil.BaseTest
	sys *osutil.SyscallRecorder
	log *bytes.Buffer
}

func (s *mountSuite) SetUpTest(c *C) {
	s.BaseTest.SetUpTest(c)
	s.sys = &osutil.SyscallRecorder{}
	s.BaseTest.AddCleanup(osutil.MockSystemCalls(s.sys))
	buf, restore := logger.MockLogger()
	s.BaseTest.AddCleanup(restore)
	s.log = buf
}

func (s *mountSuite) TearDownTest(c *C) {
	s.sys.CheckForStrayDescriptors(c)
	s.BaseTest.TearDownTest(c)
}

var _ = Suite(&mountSuite{})

func (s *mountSuite) TestIsMountedHappyish(c *C) {
	// note the different optional fields
	const content = "" +
		"44 24 7:1 / /snap/ubuntu-core/855 rw,relatime shared:27 - squashfs /dev/loop1 ro\n" +
		"44 24 7:1 / /snap/something/123 rw,relatime - squashfs /dev/loop2 ro\n" +
		"44 24 7:1 / /snap/random/456 rw,relatime opt:1 shared:27 - squashfs /dev/loop1 ro\n"
	defer osutil.MockMountInfo(content)()

	mounted, err := osutil.IsMounted("/snap/ubuntu-core/855")
	c.Check(err, IsNil)
	c.Check(mounted, Equals, true)

	mounted, err = osutil.IsMounted("/snap/something/123")
	c.Check(err, IsNil)
	c.Check(mounted, Equals, true)

	mounted, err = osutil.IsMounted("/snap/random/456")
	c.Check(err, IsNil)
	c.Check(mounted, Equals, true)

	mounted, err = osutil.IsMounted("/random/made/up/name")
	c.Check(err, IsNil)
	c.Check(mounted, Equals, false)
}

func (s *mountSuite) TestIsMountedBroken(c *C) {
	defer osutil.MockMountInfo("44 24 7:1 ...truncated-stuff")()

	mounted, err := osutil.IsMounted("/snap/ubuntu-core/855")
	c.Check(err, ErrorMatches, "incorrect number of fields, .*")
	c.Check(mounted, Equals, false)
}

// We want to mount a tmpfs over /var/tmp and everything works.
func (s *mountSuite) TestConservativeMountSuccess(c *C) {
	s.sys.InsertFstatResult(`fstat 4 <ptr>`, syscall.Stat_t{})
	before := "28 0 8:1 / / rw,relatime shared:1 - ext4 /dev/sda1 rw,errors=remount-ro,data=ordered"
	after := before + `
131 86 0:48 / /var/tmp rw,relatime shared:73 - tmpfs tmpfs rw
`
	defer osutil.MockMountInfoVary(before, after)()

	err := osutil.ConservativeMount(&osutil.MountEntry{Dir: "/var/tmp", Type: "tmpfs", Name: "tmpfs"})
	c.Assert(err, IsNil)
	c.Assert(s.sys.Calls(), DeepEquals, []string{
		`open "/" O_NOFOLLOW|O_CLOEXEC|O_DIRECTORY 0`, // -> 3
		`openat 3 "var" O_NOFOLLOW|O_CLOEXEC|O_DIRECTORY 0`, // -> 4
		`fstat 4 <ptr>`,
		`openat 4 "tmp" O_NOFOLLOW|O_CLOEXEC|O_DIRECTORY 0`, // -> 5
		`close 5`,
		`close 4`,
		`close 3`,
		`mount "tmpfs" "/var/tmp" "tmpfs" 0 ""`,
	})
}

// We want to mount tmpfs over /var/tmp but something is already mounted there.
func (s *mountSuite) TestConservativeMountCannotOvermount(c *C) {
	s.sys.InsertFstatResult(`fstat 4 <ptr>`, syscall.Stat_t{})
	defer osutil.MockMountInfo(`28 0 8:1 / / rw,relatime shared:1 - ext4 /dev/sda1 rw,errors=remount-ro,data=ordered
131 86 0:48 / /var/tmp rw,relatime shared:73 - tmpfs tmpfs rw`)()

	err := osutil.ConservativeMount(&osutil.MountEntry{Dir: "/var/tmp", Type: "tmpfs", Name: "tmpfs"})
	c.Assert(err, ErrorMatches, "cannot mount over existing mount entry .* /var/tmp .*")
	c.Assert(s.sys.Calls(), DeepEquals, []string{
		`open "/" O_NOFOLLOW|O_CLOEXEC|O_DIRECTORY 0`, // -> 3
		`openat 3 "var" O_NOFOLLOW|O_CLOEXEC|O_DIRECTORY 0`, // -> 4
		`fstat 4 <ptr>`,
		`openat 4 "tmp" O_NOFOLLOW|O_CLOEXEC|O_DIRECTORY 0`, // -> 5
		`close 5`,
		`close 4`,
		`close 3`,
	})
}

// We want to mount tmpfs over /var/tmp but someone races a symlink /var/tmp -> /evil.
func (s *mountSuite) TestConservativeMountRaceLost(c *C) {
	s.sys.InsertFstatResult(`fstat 4 <ptr>`, syscall.Stat_t{})
	before := "28 0 8:1 / / rw,relatime shared:1 - ext4 /dev/sda1 rw,errors=remount-ro,data=ordered"
	after := before + `
131 86 0:48 / /evil rw,relatime shared:73 - tmpfs tmpfs rw
`
	defer osutil.MockMountInfoVary(before, after)()

	err := osutil.ConservativeMount(&osutil.MountEntry{Dir: "/var/tmp", Type: "tmpfs", Name: "tmpfs"})
	c.Assert(err, ErrorMatches, `cannot ensure mount consistency: expected to find mounted "/var/tmp"`)
	c.Assert(s.sys.Calls(), DeepEquals, []string{
		`open "/" O_NOFOLLOW|O_CLOEXEC|O_DIRECTORY 0`, // -> 3
		`openat 3 "var" O_NOFOLLOW|O_CLOEXEC|O_DIRECTORY 0`, // -> 4
		`fstat 4 <ptr>`,
		`openat 4 "tmp" O_NOFOLLOW|O_CLOEXEC|O_DIRECTORY 0`, // -> 5
		`close 5`,
		`close 4`,
		`close 3`,
		`mount "tmpfs" "/var/tmp" "tmpfs" 0 ""`,
	})
}
