// -*- Mode: Go; indent-tabs-mode: t -*-

/*
 * Copyright (C) 2025 Canonical Ltd
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

package vfs_test

import (
	"io/fs"
	"testing"
	"testing/fstest"

	"github.com/snapcore/snapd/osutil/vfs"
)

func assertVFS(t *testing.T, v *vfs.VFS, expected string) {
	t.Helper()
	if v.String() != expected {
		t.Fatal("Unexpected VFS state", v)
	}
}

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func TestVFS_PropagationToShared(t *testing.T) {
	v := vfs.NewVFS(fstest.MapFS{
		"a": &fstest.MapFile{Mode: fs.ModeDir},
		"b": &fstest.MapFile{Mode: fs.ModeDir},
		"c": &fstest.MapFile{Mode: fs.ModeDir},
	})
	must(t, v.Mount(fstest.MapFS{"1": &fstest.MapFile{Mode: fs.ModeDir}}, "a"))
	must(t, v.MakeShared("a"))
	must(t, v.BindMount("a", "b"))
	must(t, v.BindMount("a", "c"))
	must(t, v.Mount(fstest.MapFS{}, "a/1"))
	// The mount a/1 is propagated to a, b and c.
	assertVFS(t, v, `
-1 -1 0:0 / / rw - (fstype) (source) rw
0  -1 0:0 / /a rw shared:1 - (fstype) (source) rw
1  -1 0:0 / /b rw shared:1 - (fstype) (source) rw
2  -1 0:0 / /c rw shared:1 - (fstype) (source) rw
3  0 0:0 / /a/1 rw shared:2 - (fstype) (source) rw
4  1 0:0 / /b/1 rw shared:2 - (fstype) (source) rw
5  2 0:0 / /c/1 rw shared:2 - (fstype) (source) rw
`)
}

func TestVFS_PropagationToSlave(t *testing.T) {
	v := vfs.NewVFS(fstest.MapFS{
		"a": &fstest.MapFile{Mode: fs.ModeDir},
		"b": &fstest.MapFile{Mode: fs.ModeDir},
		"c": &fstest.MapFile{Mode: fs.ModeDir},
	})
	must(t, v.Mount(fstest.MapFS{"1": &fstest.MapFile{Mode: fs.ModeDir}}, "a"))
	must(t, v.MakeShared("a"))
	must(t, v.BindMount("a", "b"))
	must(t, v.MakeSlave("b"))
	must(t, v.BindMount("a", "c"))
	must(t, v.MakeSlave("c"))
	must(t, v.Mount(fstest.MapFS{}, "a/1"))
	// The mount a/1 is propagated to a, b and c.
	assertVFS(t, v, `
-1 -1 0:0 / / rw - (fstype) (source) rw
0  -1 0:0 / /a rw shared:1 - (fstype) (source) rw
1  -1 0:0 / /b rw master:1 - (fstype) (source) rw
2  -1 0:0 / /c rw master:1 - (fstype) (source) rw
3  0 0:0 / /a/1 rw shared:2 - (fstype) (source) rw
4  1 0:0 / /b/1 rw master:2 - (fstype) (source) rw
5  2 0:0 / /c/1 rw master:2 - (fstype) (source) rw
`)
}

func TestVFS_PropagationToSharedSlave(t *testing.T) {
	v := vfs.NewVFS(fstest.MapFS{
		"a": &fstest.MapFile{Mode: fs.ModeDir},
		"b": &fstest.MapFile{Mode: fs.ModeDir},
		"c": &fstest.MapFile{Mode: fs.ModeDir},
	})
	must(t, v.Mount(fstest.MapFS{"1": &fstest.MapFile{Mode: fs.ModeDir}}, "a"))
	must(t, v.MakeShared("a"))
	must(t, v.BindMount("a", "b"))
	must(t, v.MakeSlave("b"))
	must(t, v.MakeShared("b"))
	must(t, v.BindMount("a", "c"))
	must(t, v.MakeSlave("c"))
	must(t, v.MakeShared("c"))
	must(t, v.Mount(fstest.MapFS{}, "a/1"))
	// The mount a/1 is propagated to a, b and c.
	assertVFS(t, v, `
-1 -1 0:0 / / rw - (fstype) (source) rw
0  -1 0:0 / /a rw shared:1 - (fstype) (source) rw
1  -1 0:0 / /b rw shared:2 master:1 - (fstype) (source) rw
2  -1 0:0 / /c rw shared:3 master:1 - (fstype) (source) rw
3  0 0:0 / /a/1 rw shared:4 - (fstype) (source) rw
4  1 0:0 / /b/1 rw shared:5 master:4 - (fstype) (source) rw
5  2 0:0 / /c/1 rw shared:6 master:4 - (fstype) (source) rw
`)
}

func TestVFS_MakeShared(t *testing.T) {
	t.Run("bind-keeps-sharing", func(t *testing.T) {
		v := vfs.NewVFS(fstest.MapFS{
			"a":       &fstest.MapFile{Mode: fs.ModeDir},
			"a_prime": &fstest.MapFile{Mode: fs.ModeDir},
		})
		must(t, v.Mount(fstest.MapFS{"b": &fstest.MapFile{Mode: fs.ModeDir}}, "a"))
		must(t, v.MakeShared("a"))
		must(t, v.Mount(fstest.MapFS{}, "a/b"))
		must(t, v.RecursiveBindMount("a", "a_prime"))
		assertVFS(t, v, `
-1 -1 0:0 / / rw - (fstype) (source) rw
0  -1 0:0 / /a rw shared:1 - (fstype) (source) rw
1  0 0:0 / /a/b rw shared:2 - (fstype) (source) rw
2  -1 0:0 / /a_prime rw shared:1 - (fstype) (source) rw
3  2 0:0 / /a_prime/b rw shared:2 - (fstype) (source) rw
`)
	})

	t.Run("share-propagates", func(t *testing.T) {
		v := vfs.NewVFS(fstest.MapFS{
			"a":       &fstest.MapFile{Mode: fs.ModeDir},
			"a_prime": &fstest.MapFile{Mode: fs.ModeDir},
		})
		must(t, v.Mount(fstest.MapFS{"b": &fstest.MapFile{Mode: fs.ModeDir}}, "a"))
		must(t, v.MakeShared("a"))
		must(t, v.RecursiveBindMount("a", "a_prime"))
		must(t, v.Mount(fstest.MapFS{}, "a/b"))
		assertVFS(t, v, `
-1 -1 0:0 / / rw - (fstype) (source) rw
0  -1 0:0 / /a rw shared:1 - (fstype) (source) rw
1  -1 0:0 / /a_prime rw shared:1 - (fstype) (source) rw
2  0 0:0 / /a/b rw shared:2 - (fstype) (source) rw
3  1 0:0 / /a_prime/b rw shared:2 - (fstype) (source) rw
`)
	})
}
