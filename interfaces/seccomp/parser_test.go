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

type parserSuite struct{}

var _ = Suite(&parserSuite{})

// The result may be empty list of rules.
func (s *parserSuite) TestParse0(c *C) {
	rules, err := seccomp.ParseSnippet("")
	c.Assert(err, IsNil)
	c.Assert(rules, HasLen, 0)
}

// Trivial rule is parsed correctly.
func (s *parserSuite) TestParse1(c *C) {
	rules, err := seccomp.ParseSnippet("bind")
	c.Assert(err, IsNil)
	c.Assert(rules, DeepEquals, []seccomp.Rule{{SysCall: seccomp.SysBind}})
}

// Rule can have a trailing comment on the same line.
func (s *parserSuite) TestParse2(c *C) {
	rules, err := seccomp.ParseSnippet("bind # bind is nice")
	c.Assert(err, IsNil)
	c.Assert(rules, DeepEquals, []seccomp.Rule{{Comment: "# bind is nice", SysCall: seccomp.SysBind}})
}

// Comments can precede a rule.
func (s *parserSuite) TestParse3(c *C) {
	rules, err := seccomp.ParseSnippet("# bind is nice\nbind\n")
	c.Assert(err, IsNil)
	c.Assert(rules, DeepEquals, []seccomp.Rule{{Comment: "# bind is nice", SysCall: seccomp.SysBind}})
}

// Multi-line comments are aggregated correctly.
func (s *parserSuite) TestParse4(c *C) {
	rules, err := seccomp.ParseSnippet("# bind is nice\n# bind is very very nice!\nbind\n")
	c.Assert(err, IsNil)
	c.Assert(rules, DeepEquals, []seccomp.Rule{{Comment: "# bind is nice\n# bind is very very nice!", SysCall: seccomp.SysBind}})
}

// Filtering can be done on numeric arguments.
func (s *parserSuite) TestParse5(c *C) {
	rules, err := seccomp.ParseSnippet("fchown - 0 42")
	c.Assert(err, IsNil)
	c.Assert(rules, DeepEquals, []seccomp.Rule{{
		SysCall: seccomp.SysFchown,
		Args: []seccomp.ArgConstraint{
			{Op: seccomp.Any},
			{Op: seccomp.Equal, Value: "0", ResolvedValue: 0, IsResolved: true},
			{Op: seccomp.Equal, Value: "42", ResolvedValue: 42, IsResolved: true},
		},
	}})
}

// Filtering can be done on symbolic arguments.
func (s *parserSuite) TestParse6(c *C) {
	rules, err := seccomp.ParseSnippet("socket AF_NETLINK - NETLINK_AUDIT")
	c.Assert(err, IsNil)
	c.Assert(rules, DeepEquals, []seccomp.Rule{{
		SysCall: seccomp.SysSocket,
		Args: []seccomp.ArgConstraint{
			{Op: seccomp.Equal, Value: "AF_NETLINK", ResolvedValue: syscall.AF_NETLINK, IsResolved: true},
			{Op: seccomp.Any},
			{Op: seccomp.Equal, Value: "NETLINK_AUDIT", ResolvedValue: syscall.NETLINK_AUDIT, IsResolved: true},
		},
	}})
}

// Multiple rules can be returned
func (s *parserSuite) TestParse7(c *C) {
	rules, err := seccomp.ParseSnippet("socket\nbind")
	c.Assert(err, IsNil)
	c.Assert(rules, DeepEquals, []seccomp.Rule{
		{SysCall: seccomp.SysSocket},
		{SysCall: seccomp.SysBind},
	})
}

// Pure comment rules can be used for compatibility with commented-out snippets.
func (s *parserSuite) TestParse8(c *C) {
	rules, err := seccomp.ParseSnippet("# just comment")
	c.Assert(err, IsNil)
	c.Assert(rules, DeepEquals, []seccomp.Rule{{Comment: "# just comment"}})
}
