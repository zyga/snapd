// -*- Mode: Go; indent-tabs-mode: t -*-

/*
 * Copyright (C) 2024 Canonical Ltd
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

package ctlcmd_test

import (
	"strings"
	"testing"

	. "gopkg.in/check.v1"

	"github.com/snapcore/snapd/overlord/hookstate/ctlcmd"
)

func TestCtlcmd(t *testing.T) { TestingT(t) }

type execSuite struct{}

var _ = Suite(&execSuite{})

func (s *execSuite) TestExecCommandName(c *C) {
	cmd := ctlcmd.NewExecCmd(nil, nil)
	c.Check(cmd.Name(), Equals, "exec")
}

func (s *execSuite) TestExecCommandShortHelp(c *C) {
	cmd := ctlcmd.NewExecCmd(nil, nil)
	c.Check(cmd.ShortHelp(), Not(Equals), "")
}

func (s *execSuite) TestExecCommandLongHelp(c *C) {
	cmd := ctlcmd.NewExecCmd(nil, nil)
	c.Check(cmd.LongHelp(), Not(Equals), "")
	c.Check(strings.Contains(cmd.LongHelp(), "workload"), Equals, true)
}

func (s *execSuite) TestExecCommandRequiresWorkload(c *C) {
	cmd := ctlcmd.NewExecCmd(nil, nil)
	cmd.SetWorkload("test")
	cmd.SetCommand([]string{"echo", "hello"})

	// No context available - should fail before workload validation
	err := cmd.Execute(nil)
	c.Assert(err, NotNil)
	c.Check(err.Error(), Matches, `.*cannot invoke snapctl.*`)
}

func (s *execSuite) TestExecCommandRequiresCommand(c *C) {
	cmd := ctlcmd.NewExecCmd(nil, nil)
	cmd.SetWorkload("test")
	cmd.SetCommand(nil)

	err := cmd.Execute(nil)
	c.Assert(err, NotNil)
	c.Check(err.Error(), Matches, `.*no command.*`)
}

func (s *execSuite) TestExecCommandRequiresContext(c *C) {
	cmd := ctlcmd.NewExecCmd(nil, nil)
	cmd.SetWorkload("test")
	cmd.SetCommand([]string{"echo", "hello"})

	// No context available
	err := cmd.Execute(nil)
	c.Assert(err, NotNil)
	c.Check(err.Error(), Matches, `.*cannot invoke snapctl.*`)
}

func (s *execSuite) TestExecRequestJSON(c *C) {
	req := ctlcmd.NewExecRequest("start", "my-snap", "test-workload", []string{"echo", "hello"})

	data, err := req.MarshalJSON()
	c.Assert(err, IsNil)
	c.Check(string(data), Matches, `.*"action":"start".*"snap":"my-snap".*"workload":"test-workload".*"command".*\["echo","hello"\].*`)
}

func (s *execSuite) TestExecResponseCookie(c *C) {
	resp := ctlcmd.NewExecResponse("cookie-123", "scope-test", 12345, "running")
	c.Check(resp.Cookie, Equals, "cookie-123")
	c.Check(resp.Scope, Equals, "scope-test")
	c.Check(resp.PID, Equals, 12345)
	c.Check(resp.Status, Equals, "running")
}

func (s *execSuite) TestExecStatusResponse(c *C) {
	resp := ctlcmd.NewExecStatusResponse("cookie-123", "my-snap", "test-workload", "scope-test", 12345, "done", 0)
	c.Check(resp.Cookie, Equals, "cookie-123")
	c.Check(resp.Snap, Equals, "my-snap")
	c.Check(resp.Workload, Equals, "test-workload")
	c.Check(resp.Status, Equals, "done")
	c.Check(resp.ExitCode, Equals, 0)
}
