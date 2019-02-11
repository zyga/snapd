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

package snapstate_test

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	. "gopkg.in/check.v1"

	"github.com/snapcore/snapd/dirs"
	"github.com/snapcore/snapd/overlord/snapstate"
	"github.com/snapcore/snapd/overlord/state"
	"github.com/snapcore/snapd/snap"
)

type refreshSuite struct {
	state *state.State
}

var _ = Suite(&refreshSuite{})

func (s *refreshSuite) SetUpTest(c *C) {
	s.state = state.New(nil)
}

func (s *refreshSuite) TestSoftRefreshCheck(c *C) {
	// Mock directory locations.
	dirs.SetRootDir(c.MkDir())
	defer dirs.SetRootDir("")

	s.state.Lock()
	defer s.state.Unlock()

	// Mock the presence of the foo snap
	snapstate.Set(s.state, "foo", &snapstate.SnapState{
		Active: true,
		Sequence: []*snap.SideInfo{
			{RealName: "foo", Revision: snap.R(5), SnapID: "foo-id"},
		},
		Current:  snap.R(5),
		SnapType: "app",
		UserID:   1,
	})

	// Mock the info about the foo snap.
	restore := snapstate.MockSnapReadInfo(func(name string, si *snap.SideInfo) (*snap.Info, error) {
		if name != "foo" {
			panic("expected only foo snap")
		}

		info := &snap.Info{
			SideInfo: *si,
			Type:     snap.TypeApp,
		}
		info.Apps = map[string]*snap.AppInfo{
			"daemon": {
				Snap:   info,
				Name:   "daemon",
				Daemon: "simple",
			},
			"app": {
				Snap: info,
				Name: "app",
			},
		}
		info.Hooks = map[string]*snap.HookInfo{
			"configure": {
				Snap: info,
				Name: "configure",
			},
		}
		return info, nil
	})
	defer restore()

	// There are no errors when directories are absent.
	err := snapstate.SoftRefreshCheck(s.state, "foo")
	c.Check(err, IsNil)

	writePids := func(dir string, pids []int) {
		err := os.MkdirAll(dir, 0755)
		c.Assert(err, IsNil)
		var buf bytes.Buffer
		for _, pid := range pids {
			fmt.Fprintf(&buf, "%d\n", pid)
		}
		err = ioutil.WriteFile(filepath.Join(dir, "cgroup.procs"), buf.Bytes(), 0644)
		c.Assert(err, IsNil)
	}

	snapPath := filepath.Join(dirs.FreezerCgroupDir, "snap.foo")
	daemonPath := filepath.Join(dirs.PidsCgroupDir, "snap.foo.daemon")
	appPath := filepath.Join(dirs.PidsCgroupDir, "snap.foo.app")
	hookPath := filepath.Join(dirs.PidsCgroupDir, "snap.foo.hooks.configure")

	// Processes not traced to a service block refresh.
	writePids(snapPath, []int{100})
	err = snapstate.SoftRefreshCheck(s.state, "foo")
	c.Check(err, ErrorMatches, `snap "foo" has running apps or hooks`)

	// Services are excluded from the check.
	writePids(snapPath, []int{100})
	writePids(daemonPath, []int{100})
	err = snapstate.SoftRefreshCheck(s.state, "foo")
	c.Check(err, IsNil)

	// Apps are not excluded.
	writePids(snapPath, []int{100, 101})
	writePids(daemonPath, []int{100})
	writePids(appPath, []int{101})
	err = snapstate.SoftRefreshCheck(s.state, "foo")
	c.Check(err, ErrorMatches, `snap "foo" has running apps or hooks`)
	c.Check(err.(*snapstate.BusySnapError).Pids(), DeepEquals, []int{101})

	// Hooks are not excluded.
	writePids(snapPath, []int{105})
	writePids(daemonPath, []int{})
	writePids(appPath, []int{})
	writePids(hookPath, []int{105})
	err = snapstate.SoftRefreshCheck(s.state, "foo")
	c.Check(err, ErrorMatches, `snap "foo" has running apps or hooks`)
	c.Check(err.(*snapstate.BusySnapError).Pids(), DeepEquals, []int{105})
}
