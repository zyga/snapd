// -*- Mode: Go; indent-tabs-mode: t -*-

/*
 * Copyright (C) 2026 Canonical Ltd
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

package backend_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	. "gopkg.in/check.v1"

	"github.com/snapcore/snapd/dirs"
	"github.com/snapcore/snapd/logger"
	"github.com/snapcore/snapd/snap"
	"github.com/snapcore/snapd/testutil"
)

// mockSnapDiscardNs sets up a fake snap-discard-ns binary in a temp directory
// and configures dirs.DistroLibExecDir so that InternalToolPath resolves to it.
func (s *snapshotSuite) mockSnapDiscardNs(c *C, exitCode int) *testutil.MockCmd {
	script := fmt.Sprintf("exit %d", exitCode)
	cmd := testutil.MockCommand(c, "snap-discard-ns", script)
	s.restore = append(s.restore, func() {
		dirs.DistroLibExecDir = "/usr/lib/snapd"
	})
	dirs.DistroLibExecDir = cmd.BinDir()
	return cmd
}

// createPreservedNsFile creates the preserved namespace file for the given snap.
func (s *snapshotSuite) createPreservedNsFile(c *C, snapName string) {
	nsDir := dirs.SnapRunNsDir
	c.Assert(os.MkdirAll(nsDir, 0755), IsNil)
	nsPath := filepath.Join(nsDir, snapName+".mnt")
	c.Assert(os.WriteFile(nsPath, []byte("preserved"), 0644), IsNil)
}

// removePreservedNsFile removes the preserved namespace file for the given snap.
func (s *snapshotSuite) removePreservedNsFile(c *C, snapName string) {
	nsPath := filepath.Join(dirs.SnapRunNsDir, snapName+".mnt")
	os.Remove(nsPath)
}

// TestMoveFileDiscardsNamespaceBeforeRename verifies that moveFile calls
// discardPreservedNamespace (via snap-discard-ns binary) before renaming away
// an existing $SNAP_DATA/<rev> directory. This is the fix for LP#2121446:
// when a snapshot restore replaces an existing snap data directory, any
// preserved mount namespace must be discarded because layout bind mounts
// (e.g., $SNAP_DATA/root -> /root) become stale after their source is renamed.
func (s *snapshotSuite) TestMoveFileDiscardsNamespaceBeforeRename(c *C) {
	if os.Geteuid() == 0 {
		c.Skip("this test cannot run as root (runuser will fail)")
	}

	shr := saveAndOpenSnapshot(c)
	defer shr.Close()

	snapName := "hello-snap"
	discardCmd := s.mockSnapDiscardNs(c, 0)
	defer discardCmd.Restore()

	// Create a preserved namespace file for the snap. This simulates what
	// snap-confine would have created when the snap was previously run.
	s.createPreservedNsFile(c, snapName)
	defer s.removePreservedNsFile(c, snapName)

	rs, err := shr.Restore(context.TODO(), snap.R(0), nil, logger.Debugf, nil)
	c.Assert(err, IsNil)
	rs.Cleanup()

	// Restore should call discardPreservedNamespace for each of the 4 data dirs
	// (system common, system rev, user common, user rev). Each calls moveFile
	// which invokes snap-discard-ns when a preserved namespace exists.
	c.Check(discardCmd.Calls(), HasLen, 4)
	for _, call := range discardCmd.Calls() {
		c.Check(call[0], Equals, "snap-discard-ns")
		c.Check(call[1], Equals, snapName)
	}
}

// TestMoveFileNoNamespaceToDiscard verifies that moveFile does NOT call
// snap-discard-ns when no preserved namespace file exists for the snap.
func (s *snapshotSuite) TestMoveFileNoNamespaceToDiscard(c *C) {
	if os.Geteuid() == 0 {
		c.Skip("this test cannot run as root (runuser will fail)")
	}

	shr := saveAndOpenSnapshot(c)
	defer shr.Close()

	discardCmd := s.mockSnapDiscardNs(c, 0)
	defer discardCmd.Restore()

	// Ensure NO preserved namespace file exists.
	s.removePreservedNsFile(c, "hello-snap")

	rs, err := shr.Restore(context.TODO(), snap.R(0), nil, logger.Debugf, nil)
	c.Assert(err, IsNil)
	rs.Cleanup()

	// snap-discard-ns should NOT have been called.
	c.Check(discardCmd.Calls(), HasLen, 0)
}

// TestMoveFileNamespaceDiscardFailureIsNonFatal verifies that if snap-discard-ns
// fails (non-zero exit), the restore operation still proceeds (discard failure
// is logged but does not abort the restore). This matches the design: discard
// is best-effort since services are already stopped during snapshot restore.
func (s *snapshotSuite) TestMoveFileNamespaceDiscardFailureIsNonFatal(c *C) {
	if os.Geteuid() == 0 {
		c.Skip("this test cannot run as root (runuser will fail)")
	}

	shr := saveAndOpenSnapshot(c)
	defer shr.Close()

	snapName := "hello-snap"
	discardCmd := s.mockSnapDiscardNs(c, 1)
	defer discardCmd.Restore()

	s.createPreservedNsFile(c, snapName)
	defer s.removePreservedNsFile(c, snapName)

	// Restore should still succeed despite the discard failure.
	rs, err := shr.Restore(context.TODO(), snap.R(0), nil, logger.Debugf, nil)
	c.Assert(err, IsNil)
	rs.Cleanup()
}
