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

package main_test

import (
	. "gopkg.in/check.v1"

	update "github.com/snapcore/snapd/cmd/snap-update-ns"
	"github.com/snapcore/snapd/osutil"
)

func (s *changeSuite) TestContentLayoutInitiallyConnectedNoChanges(c *C) {
	current, err := osutil.LoadMountProfile("testdata/content-layout-1-initially-connected.current.fstab")
	c.Assert(err, IsNil)
	desired, err := osutil.LoadMountProfile("testdata/content-layout-1-initially-connected.desired.fstab")
	c.Assert(err, IsNil)
	changes := update.NeededChanges(current, desired)
	showCurrentDesiredAndChanges(c, current, desired, changes)

	// The change plan is to do nothing.
	// Note that the order of entries is back to front.
	//
	// At this time, the mount namespace is correct:
	// zyga@wyzima:/run/snapd/ns$ sudo nsenter -mtest-snapd-content-layout.mnt
	// root@wyzima:/# ls -l /usr/share/secureboot/potato
	// total 1
	// -rw-rw-r-- 1 root root 22 Aug 30 09:36 canary.txt
	// drwxrwxr-x 2 root root 32 Aug 30 09:36 meta
	// root@wyzima:/# ls -l /snap/test-snapd-content-layout/
	// current/ x1/      x2/
	// root@wyzima:/# ls -l /snap/test-snapd-content-layout/x2/attached-content/
	// total 1
	// -rw-rw-r-- 1 root root 22 Aug 30 09:36 canary.txt
	// drwxrwxr-x 2 root root 32 Aug 30 09:36 meta
	c.Assert(changes, DeepEquals, []*update.Change{
		&update.Change{Action: "keep", Entry: current.Entries[4]},
		&update.Change{Action: "keep", Entry: current.Entries[3]},
		&update.Change{Action: "keep", Entry: current.Entries[2]},
		&update.Change{Action: "keep", Entry: current.Entries[1]},
		&update.Change{Action: "keep", Entry: current.Entries[0]},
	})
}

func (s *changeSuite) TestContentLayoutNowDisconnected(c *C) {
	current, err := osutil.LoadMountProfile("testdata/content-layout-1-initially-connected.current.fstab")
	c.Assert(err, IsNil)
	desired, err := osutil.LoadMountProfile("testdata/content-layout-2-after-disconnect.desired.fstab")
	c.Assert(err, IsNil)
	changes := update.NeededChanges(current, desired)
	showCurrentDesiredAndChanges(c, current, desired, changes)

	// The change plan is to do detach the content entry.
	//
	// Detached entries are first isolated from mount propagation. So the bug
	// here is that the mount entry propagated to the layout during initial
	// construction sticks around and is not updated. This is a bug.
	// This is tracked as https://warthogs.atlassian.net/browse/SNAPDENG-31645
	//
	// zyga@wyzima:/run/snapd/ns$ sudo nsenter -mtest-snapd-content-layout.mnt
	// root@wyzima:/# ls -l /snap/test-snapd-content-layout/x2/attached-content/
	// total 1
	// -rw-rw-r-- 1 root root 46 Aug 30 09:36 hidden
	// root@wyzima:/# ls -l /usr/share/secureboot/potato
	// total 1
	// -rw-rw-r-- 1 root root 22 Aug 30 09:36 canary.txt
	// drwxrwxr-x 2 root root 32 Aug 30 09:36 meta
	//
	// Note that the order of entries is back to front. There is another bug
	// here, although it is not visible from the change plan alone. The reverse
	// order of mount entries listed here is actually stored as the new current
	// mount profile. This is tracked as
	// https://warthogs.atlassian.net/browse/SNAPDENG-31644
	c.Assert(changes, DeepEquals, []*update.Change{
		&update.Change{Action: "unmount", Entry: withDetachOption(current.Entries[4])},
		&update.Change{Action: "keep", Entry: current.Entries[3]},
		&update.Change{Action: "keep", Entry: current.Entries[2]},
		&update.Change{Action: "unmount", Entry: withDetachOption(current.Entries[1])},
		&update.Change{Action: "keep", Entry: current.Entries[0]},
		&update.Change{Action: "mount", Entry: current.Entries[4]},
	})

	// - change: unmount (/snap/test-snapd-content-layout/x2/attached-content /usr/share/secureboot/potato none rbind,rw,x-snapd.origin=layout,x-snapd.detach 0 0)
	// - change: keep (/usr/share/secureboot/updates /usr/share/secureboot/updates none rbind,x-snapd.synthetic,x-snapd.needed-by=/usr/share/secureboot/potato,x-snapd.detach 0 0)
	// - change: keep (tmpfs /usr/share/secureboot tmpfs x-snapd.synthetic,x-snapd.needed-by=/usr/share/secureboot/potato,mode=0755,uid=0,gid=0 0 0)
	// - change: unmount (/snap/test-snapd-content/x1 /snap/test-snapd-content-layout/x2/attached-content none bind,ro,x-snapd.detach 0 0)
	// - change: keep (tmpfs / tmpfs x-snapd.origin=rootfs 0 0)
	// - change: mount (/snap/test-snapd-content-layout/x2/attached-content /usr/share/secureboot/potato none rbind,rw,x-snapd.origin=layout 0 0)
}

func (s *changeSuite) TestContentLayoutThenReconnected(c *C) {
	current, err := osutil.LoadMountProfile("testdata/content-layout-2-after-disconnect.current.fstab")
	c.Assert(err, IsNil)
	desired, err := osutil.LoadMountProfile("testdata/content-layout-3-after-reconnect.desired.fstab")
	c.Assert(err, IsNil)
	changes := update.NeededChanges(current, desired)
	showCurrentDesiredAndChanges(c, current, desired, changes)

	// In theory we should get back to the initial state but the reality is
	// much more complicated. The change looks good on paper but the
	// propagation that is not taken into account makes the actual mount
	// namespace incorrect. The content connection is new and correct but the layout
	// is still the same and was not propagated.
	//
	// zyga@wyzima:/run/snapd/ns$ sudo nsenter -mtest-snapd-content-layout.mnt
	// root@wyzima:/# ls -l /usr/share/secureboot/potato
	// total 1
	// -rw-rw-r-- 1 root root 22 Aug 30 09:36 canary.txt
	// drwxrwxr-x 2 root root 32 Aug 30 09:36 meta
	// root@wyzima:/# ls -l /snap/test-snapd-content-layout/x2/attached-content/
	// total 1
	// -rw-rw-r-- 1 root root 22 Aug 30 09:36 canary.txt
	// drwxrwxr-x 2 root root 32 Aug 30 09:36 meta
	//
	// Yes, but:
	//
	// root@wyzima:/# cat /proc/self/mountinfo  | grep attached
	// 212 945 7:12 / /snap/test-snapd-content-layout/x2/attached-content ro,relatime master:34 - squashfs /dev/loop12 ro,errors=continue,threads=single
	//
	// root@wyzima:/# cat /proc/self/mountinfo  | grep potato
	// 572 598 7:12 / /usr/share/secureboot/potato ro,relatime master:34 - squashfs /dev/loop12 ro,errors=continue,threads=single
	c.Assert(changes, DeepEquals, []*update.Change{
		&update.Change{Action: "keep", Entry: current.Entries[3]},
		&update.Change{Action: "keep", Entry: current.Entries[2]},
		&update.Change{Action: "keep", Entry: current.Entries[1]},
		&update.Change{Action: "keep", Entry: current.Entries[0]},
		&update.Change{Action: "mount", Entry: desired.Entries[1]},
	})

	// The actual entry for clarity.
	c.Assert(changes[4].Entry, DeepEquals, osutil.MountEntry{
		Name:    "/snap/test-snapd-content/x1",
		Dir:     "/snap/test-snapd-content-layout/x2/attached-content",
		Type:    "none",
		Options: []string{"bind", "ro"},
	})
}

func withDetachOption(e osutil.MountEntry) osutil.MountEntry {
	e.Options = append([]string{}, e.Options...)
	e.Options = append(e.Options, "x-snapd.detach")
	return e
}

func showCurrentDesiredAndChanges(c *C, current, desired *osutil.MountProfile, changes []*update.Change) {
	c.Logf("Number of current entires: %d", len(current.Entries))
	for _, entry := range current.Entries {
		c.Logf("- current : %v", entry)
	}
	c.Logf("Number of desired entires: %d", len(desired.Entries))
	for _, entry := range desired.Entries {
		c.Logf("- desired: %v", entry)
	}
	c.Logf("Number of changes: %d", len(changes))
	for _, change := range changes {
		c.Logf("- change: %v", change)
	}
}
