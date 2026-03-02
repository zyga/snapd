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

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	. "gopkg.in/check.v1"

	"github.com/snapcore/snapd/client"
	"github.com/snapcore/snapd/testutil"
)

func TestT(t *testing.T) { TestingT(t) }

type snapctlSuite struct {
	server            *httptest.Server
	oldArgs           []string
	expectedContextID string
	expectedArgs      []string
	expectedStdin     []byte
	restoreEnv        func()
	restoreAuthEnv    func()
}

var _ = Suite(&snapctlSuite{})

func (s *snapctlSuite) SetUpTest(c *C) {
	s.restoreEnv = testutil.MockEnv(map[string]string{
		"SNAP_COOKIE":  "snap-context-test",
		"SNAP_CONTEXT": "",
	})
	// don't use SNAP_CONTEXT, in case other tests accidentally leak this
	n := 0
	s.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch n {
		case 0:
			c.Assert(r.Method, Equals, "POST")
			c.Assert(r.URL.Path, Equals, "/v2/snapctl")
			c.Assert(r.Header.Get("Authorization"), Equals, "")

			var snapctlPostData client.SnapCtlPostData
			decoder := json.NewDecoder(r.Body)
			c.Assert(decoder.Decode(&snapctlPostData), IsNil)
			c.Assert(snapctlPostData.ContextID, Equals, s.expectedContextID)
			c.Assert(snapctlPostData.Args, DeepEquals, s.expectedArgs)
			c.Assert(snapctlPostData.Stdin, DeepEquals, s.expectedStdin)

			fmt.Fprintln(w, `{"type": "sync", "result": {"stdout": "test stdout", "stderr": "test stderr"}}`)
		default:
			c.Fatalf("expected to get 1 request, now on %d", n+1)
		}

		n++
	}))
	clientConfig.BaseURL = s.server.URL
	s.oldArgs = os.Args
	os.Args = []string{"snapctl"}
	s.expectedContextID = "snap-context-test"
	s.expectedArgs = []string{}

	fakeAuthPath := filepath.Join(c.MkDir(), "auth.json")
	s.restoreAuthEnv = testutil.MockEnv(map[string]string{"SNAPD_AUTH_DATA_FILENAME": fakeAuthPath})
	err := os.WriteFile(fakeAuthPath, []byte(`{"macaroon":"user-macaroon"}`), 0644)
	c.Assert(err, IsNil)
}

func (s *snapctlSuite) TearDownTest(c *C) {
	s.restoreAuthEnv()
	s.restoreEnv()
	clientConfig.BaseURL = ""
	s.server.Close()
	os.Args = s.oldArgs
}

func (s *snapctlSuite) TestSnapctl(c *C) {
	stdout, stderr, err := run(nil)
	c.Check(err, IsNil)
	c.Check(string(stdout), Equals, "test stdout")
	c.Check(string(stderr), Equals, "test stderr")
}

func (s *snapctlSuite) TestSnapctlWithArgs(c *C) {
	os.Args = []string{"snapctl", "foo", "--bar"}

	s.expectedArgs = []string{"foo", "--bar"}
	stdout, stderr, err := run(nil)
	c.Check(err, IsNil)
	c.Check(string(stdout), Equals, "test stdout")
	c.Check(string(stderr), Equals, "test stderr")
}

func (s *snapctlSuite) TestSnapctlHelp(c *C) {
	restore := testutil.MockEnv(map[string]string{"SNAP_COOKIE": ""})
	defer restore()
	s.expectedContextID = ""

	os.Args = []string{"snapctl", "-h"}
	s.expectedArgs = []string{"-h"}

	_, _, err := run(nil)
	c.Check(err, IsNil)
}

func (s *snapctlSuite) TestSnapctlWithStdin(c *C) {
	s.expectedStdin = []byte("hello")
	mockStdin := bytes.NewBuffer(s.expectedStdin)

	_, _, err := run(mockStdin)
	c.Check(err, IsNil)
}
