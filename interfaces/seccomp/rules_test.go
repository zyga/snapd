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

package seccomp_test

import (
	"syscall"

	. "gopkg.in/check.v1"

	"github.com/snapcore/snapd/interfaces/seccomp"
)

type ruleSuite struct{}

var _ = Suite(&ruleSuite{})

func (s *ruleSuite) TestString1(c *C) {
	r := seccomp.Rule{SysCall: seccomp.SysBind}
	c.Assert(r.String(), Equals, "bind\n")
}

func (s *ruleSuite) TestString2(c *C) {
	r := seccomp.Rule{
		SysCall: seccomp.SysSocket,
		Args: []seccomp.ArgConstraint{
			{Op: seccomp.Equal, Value: "AF_NETLINK", ResolvedValue: syscall.AF_NETLINK, IsResolved: true},
			{Op: seccomp.Any},
			{Op: seccomp.Equal, Value: "NETLINK_CONNECTOR", ResolvedValue: syscall.NETLINK_CONNECTOR, IsResolved: true},
		},
	}
	c.Assert(r.String(), Equals, "# resolved by snapd, was: socket AF_NETLINK - NETLINK_CONNECTOR\nsocket 16 - 11\n")
}

func (s *ruleSuite) TestString3(c *C) {
	r := seccomp.Rule{SysCall: seccomp.SysBind, Comment: "# Allow usage of bind"}
	c.Assert(r.String(), Equals, "# Allow usage of bind\nbind\n")
}

func (s *ruleSuite) TestString4(c *C) {
	r := seccomp.Rule{
		SysCall: seccomp.SysNice,
		Args:    []seccomp.ArgConstraint{{Op: seccomp.LessEqual, Value: "19"}},
	}
	c.Assert(r.String(), Equals, "nice <=19\n")
}

func (s *ruleSuite) TestConstraintOps(c *C) {
	c.Assert(seccomp.Any.String(), Equals, "-")
	c.Assert(seccomp.Equal.String(), Equals, "")
	c.Assert(seccomp.NotEqual.String(), Equals, "!")
	c.Assert(seccomp.GreaterEqual.String(), Equals, ">=")
	c.Assert(seccomp.LessEqual.String(), Equals, "<=")
	c.Assert(seccomp.Greater.String(), Equals, ">")
	c.Assert(seccomp.Less.String(), Equals, "<")
	c.Assert(seccomp.Mask.String(), Equals, "|")
	c.Assert(seccomp.ConstraintOp(1000).String, PanicMatches,
		`unexpected seccomp argument constraint operator 1000`)
}

func (s *ruleSuite) TestArgConstraints(c *C) {
	c.Assert(seccomp.ArgConstraint{Op: seccomp.Any, Value: "value"}.String(), Equals, "-")
	c.Assert(seccomp.ArgConstraint{Op: seccomp.Equal, Value: "value"}.String(), Equals, "value")
	c.Assert(seccomp.ArgConstraint{Op: seccomp.NotEqual, Value: "value"}.String(), Equals, "!value")
	c.Assert(seccomp.ArgConstraint{Op: seccomp.GreaterEqual, Value: "value"}.String(), Equals, ">=value")
	c.Assert(seccomp.ArgConstraint{Op: seccomp.LessEqual, Value: "value"}.String(), Equals, "<=value")
	c.Assert(seccomp.ArgConstraint{Op: seccomp.Greater, Value: "value"}.String(), Equals, ">value")
	c.Assert(seccomp.ArgConstraint{Op: seccomp.Less, Value: "value"}.String(), Equals, "<value")
	c.Assert(seccomp.ArgConstraint{Op: seccomp.Mask, Value: "value"}.String(), Equals, "|value")
}
