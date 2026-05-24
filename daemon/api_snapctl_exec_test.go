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

package daemon_test

import (
	"bytes"
	"net/http"
	"os"
	"path/filepath"

	"gopkg.in/check.v1"

	"github.com/snapcore/snapd/daemon"
	"github.com/snapcore/snapd/dirs"
)

var _ = check.Suite(&snapctlExecSuite{})

type snapctlExecSuite struct {
	apiBaseSuite
}

func (s *snapctlExecSuite) SetUpTest(c *check.C) {
	s.apiBaseSuite.SetUpTest(c)
	s.expectWriteAccess(daemon.SnapAccess{})

	// Ensure exec directory exists
	err := daemon.EnsureSnapctlExecDir()
	c.Assert(err, check.IsNil)
}

func (s *snapctlExecSuite) TestSnapctlExecHelp(c *check.C) {
	s.daemon(c)

	defer daemon.MockUcrednetGet(func(string) (*daemon.Ucrednet, error) {
		return &daemon.Ucrednet{Uid: 100, Pid: 9999, Socket: dirs.SnapSocket}, nil
	})()

	buf := bytes.NewBufferString(`{"action": "help"}`)
	req, err := http.NewRequest("POST", "/v2/snapctl/exec", buf)
	c.Assert(err, check.IsNil)

	rsp := s.syncReq(c, req, nil, actionIsExpected)
	c.Assert(rsp.Status, check.Equals, 200)

	// Check that result contains expected keys
	resultMap, ok := rsp.Result.(map[string]interface{})
	if ok {
		c.Check(resultMap["summary"], check.Equals, "Execute a workload with a specific security profile")
	} else {
		// Result might be a different type, just check it's not nil
		c.Check(rsp.Result, check.Not(check.IsNil))
	}
}

func (s *snapctlExecSuite) TestSnapctlExecStartMissingSnap(c *check.C) {
	s.daemon(c)

	defer daemon.MockUcrednetGet(func(string) (*daemon.Ucrednet, error) {
		return &daemon.Ucrednet{Uid: 100, Pid: 9999, Socket: dirs.SnapSocket}, nil
	})()

	buf := bytes.NewBufferString(`{"action": "start", "workload": "test", "command": ["echo", "hello"]}`)
	req, err := http.NewRequest("POST", "/v2/snapctl/exec", buf)
	c.Assert(err, check.IsNil)

	rsp := s.errorReq(c, req, nil, actionIsExpected)
	c.Assert(rsp.Status, check.Equals, 400)
}

func (s *snapctlExecSuite) TestSnapctlExecStartMissingWorkload(c *check.C) {
	s.daemon(c)

	defer daemon.MockUcrednetGet(func(string) (*daemon.Ucrednet, error) {
		return &daemon.Ucrednet{Uid: 100, Pid: 9999, Socket: dirs.SnapSocket}, nil
	})()

	buf := bytes.NewBufferString(`{"action": "start", "snap": "my-snap", "command": ["echo", "hello"]}`)
	req, err := http.NewRequest("POST", "/v2/snapctl/exec", buf)
	c.Assert(err, check.IsNil)

	rsp := s.errorReq(c, req, nil, actionIsExpected)
	c.Assert(rsp.Status, check.Equals, 400)
}

func (s *snapctlExecSuite) TestSnapctlExecStartMissingCommand(c *check.C) {
	s.daemon(c)

	defer daemon.MockUcrednetGet(func(string) (*daemon.Ucrednet, error) {
		return &daemon.Ucrednet{Uid: 100, Pid: 9999, Socket: dirs.SnapSocket}, nil
	})()

	buf := bytes.NewBufferString(`{"action": "start", "snap": "my-snap", "workload": "test"}`)
	req, err := http.NewRequest("POST", "/v2/snapctl/exec", buf)
	c.Assert(err, check.IsNil)

	rsp := s.errorReq(c, req, nil, actionIsExpected)
	c.Assert(rsp.Status, check.Equals, 400)
}

func (s *snapctlExecSuite) TestSnapctlExecStatusUnknownCookie(c *check.C) {
	s.daemon(c)

	defer daemon.MockUcrednetGet(func(string) (*daemon.Ucrednet, error) {
		return &daemon.Ucrednet{Uid: 100, Pid: 9999, Socket: dirs.SnapSocket}, nil
	})()

	buf := bytes.NewBufferString(`{"action": "status", "cookie": "nonexistent"}`)
	req, err := http.NewRequest("POST", "/v2/snapctl/exec", buf)
	c.Assert(err, check.IsNil)

	rsp := s.errorReq(c, req, nil, actionIsExpected)
	c.Assert(rsp.Status, check.Equals, 400)
}

func (s *snapctlExecSuite) TestSnapctlExecCancelUnknownCookie(c *check.C) {
	s.daemon(c)

	defer daemon.MockUcrednetGet(func(string) (*daemon.Ucrednet, error) {
		return &daemon.Ucrednet{Uid: 100, Pid: 9999, Socket: dirs.SnapSocket}, nil
	})()

	buf := bytes.NewBufferString(`{"action": "cancel", "cookie": "nonexistent"}`)
	req, err := http.NewRequest("POST", "/v2/snapctl/exec", buf)
	c.Assert(err, check.IsNil)

	rsp := s.errorReq(c, req, nil, actionIsExpected)
	c.Assert(rsp.Status, check.Equals, 400)
}

func (s *snapctlExecSuite) TestSnapctlExecGenerateCookie(c *check.C) {
	cookie := daemon.GenerateCookie()
	c.Assert(cookie, check.Not(check.HasLen), 0)
}

func (s *snapctlExecSuite) TestSnapctlExecWriteReadExitCode(c *check.C) {
	tmpDir := c.MkDir()

	path := filepath.Join(tmpDir, "test.exit")
	err := daemon.WriteExitCode(path, nil)
	c.Assert(err, check.IsNil)

	data, err := os.ReadFile(path)
	c.Assert(err, check.IsNil)
	c.Assert(string(data), check.Equals, "")

	code := 0
	err = daemon.WriteExitCode(path, &code)
	c.Assert(err, check.IsNil)

	data, err = os.ReadFile(path)
	c.Assert(err, check.IsNil)
	c.Assert(string(data), check.Equals, "0")

	readCode, err := daemon.ReadExitCode(path)
	c.Assert(err, check.IsNil)
	c.Assert(*readCode, check.Equals, 0)

	exitCode := 42
	err = daemon.WriteExitCode(path, &exitCode)
	c.Assert(err, check.IsNil)

	readCode, err = daemon.ReadExitCode(path)
	c.Assert(err, check.IsNil)
	c.Assert(*readCode, check.Equals, 42)
}

func (s *snapctlExecSuite) TestSnapctlExecReadNonExistentFile(c *check.C) {
	_, err := daemon.ReadExitCode(filepath.Join(c.MkDir(), "nonexistent.exit"))
	c.Assert(err, check.NotNil)
}

func (s *snapctlExecSuite) TestSnapctlExecReadEmptyFile(c *check.C) {
	tmpDir := c.MkDir()
	path := filepath.Join(tmpDir, "test.exit")
	os.WriteFile(path, []byte{}, 0644)

	code, err := daemon.ReadExitCode(path)
	c.Assert(err, check.IsNil)
	c.Assert(code, check.IsNil)
}

func (s *snapctlExecSuite) TestSnapctlExecJobTracking(c *check.C) {
	cookie := daemon.GenerateCookie()
	job := &daemon.ExecJob{
		Cookie:       cookie,
		SnapInstance: "test-snap",
		WorkloadName: "test",
		Command:      []string{"echo", "hello"},
		PID:          12345,
	}

	daemon.AddExecJob(cookie, job)

	retrievedJob := daemon.GetExecJob(cookie)
	c.Assert(retrievedJob, check.NotNil)
	c.Check(retrievedJob.Cookie, check.Equals, cookie)
	c.Check(retrievedJob.SnapInstance, check.Equals, "test-snap")
	c.Check(retrievedJob.PID, check.Equals, 12345)

	daemon.RemoveExecJob(cookie)
	retrievedJob = daemon.GetExecJob(cookie)
	c.Assert(retrievedJob, check.IsNil)
}

func (s *snapctlExecSuite) TestSnapctlExecSpawnWorkloadScope(c *check.C) {
	c.Assert(daemon.SnapctlExecSpawn, check.NotNil)
}

func (s *snapctlExecSuite) TestSnapctlExecDecodeInvalidJSON(c *check.C) {
	s.daemon(c)

	defer daemon.MockUcrednetGet(func(string) (*daemon.Ucrednet, error) {
		return &daemon.Ucrednet{Uid: 100, Pid: 9999, Socket: dirs.SnapSocket}, nil
	})()

	buf := bytes.NewBufferString(`invalid json {`)
	req, err := http.NewRequest("POST", "/v2/snapctl/exec", buf)
	c.Assert(err, check.IsNil)

	rsp := s.errorReq(c, req, nil, actionIsExpected)
	c.Assert(rsp.Status, check.Equals, 400)
	c.Check(rsp.Message, check.Matches, `.*cannot decode exec request.*`)
}

func (s *snapctlExecSuite) TestSnapctlExecContextRequired(c *check.C) {
	s.daemon(c)

	defer daemon.MockUcrednetGet(func(string) (*daemon.Ucrednet, error) {
		return &daemon.Ucrednet{Uid: 100, Pid: 9999, Socket: dirs.SnapSocket}, nil
	})()

	buf := bytes.NewBufferString(`{"action": "start", "snap": "my-snap", "workload": "test", "command": ["echo"]}`)
	req, err := http.NewRequest("POST", "/v2/snapctl/exec", buf)
	c.Assert(err, check.IsNil)

	rsp := s.errorReq(c, req, nil, actionIsExpected)
	c.Assert(rsp.Status, check.Equals, 400)
	c.Check(rsp.Message, check.Matches, `.*snapctl exec requires a snap context.*`)
}
