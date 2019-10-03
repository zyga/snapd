// -*- Mode: Go; indent-tabs-mode: t -*-

/*
 * Copyright (C) 2019 Canonical Ltd
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
	"testing"

	. "gopkg.in/check.v1"

	agent "github.com/snapcore/snapd/cmd/snapd-release-agent"
)

func Test(t *testing.T) { TestingT(t) }

type mainSuite struct{}

var _ = Suite(&mainSuite{})

func (s *mainSuite) TestSplitCgroupPath(c *C) {
	_, _, err := agent.SplitCgroupPath("foo")
	c.Check(err, ErrorMatches, "cgroup path is not absolute")
	_, _, err = agent.SplitCgroupPath("/foo")
	c.Check(err, ErrorMatches, "cgroup path unrelated to snaps")
	_, _, err = agent.SplitCgroupPath("/snap.foo.bar/stuff")
	c.Check(err, ErrorMatches, "cgroup path describes sub-hierarchy")
	_, _, err = agent.SplitCgroupPath("/snap.pkg")
	c.Check(err, ErrorMatches, "cgroup path is not a snap security tag")
	_, _, err = agent.SplitCgroupPath("/snap.pkg.hook.configure.wat")
	c.Check(err, ErrorMatches, "cgroup path is not a snap security tag")

	snapName, snapSecurityTag, err := agent.SplitCgroupPath("/snap.pkg.app")
	c.Check(err, IsNil)
	c.Check(snapName, Equals, "pkg")
	c.Check(snapSecurityTag, Equals, "snap.pkg.app")

	snapName, snapSecurityTag, err = agent.SplitCgroupPath("/snap.pkg.hooks.configure")
	c.Check(err, IsNil)
	c.Check(snapName, Equals, "pkg")
	c.Check(snapSecurityTag, Equals, "snap.pkg.hooks.configure")
}
