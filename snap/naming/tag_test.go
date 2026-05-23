// -*- Mode: Go; indent-tabs-mode: t -*-

/*
 * Copyright (C) 2020 Canonical Ltd
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

package naming_test

import (
	. "gopkg.in/check.v1"

	"github.com/snapcore/snapd/snap/naming"
)

type tagSuite struct{}

var _ = Suite(&tagSuite{})

func (s *tagSuite) TestParseSecurityTag(c *C) {
	// valid snap names, snap instances, app names and hook names are accepted.
	tag, err := naming.ParseSecurityTag("snap.pkg.app")
	c.Assert(err, IsNil)
	c.Check(tag.String(), Equals, "snap.pkg.app")
	c.Check(tag.InstanceName(), Equals, "pkg")
	c.Check(tag.(naming.AppSecurityTag).AppName(), Equals, "app")

	tag, err = naming.ParseSecurityTag("snap.pkg_key.app")
	c.Assert(err, IsNil)
	c.Check(tag.String(), Equals, "snap.pkg_key.app")
	c.Check(tag.InstanceName(), Equals, "pkg_key")
	c.Check(tag.(naming.AppSecurityTag).AppName(), Equals, "app")

	tag, err = naming.ParseSecurityTag("snap.pkg.hook.configure")
	c.Assert(err, IsNil)
	c.Check(tag.String(), Equals, "snap.pkg.hook.configure")
	c.Check(tag.InstanceName(), Equals, "pkg")
	c.Check(tag.(naming.HookSecurityTag).HookName(), Equals, "configure")
	c.Check(tag.(naming.HookSecurityTag).ComponentName(), Equals, "")

	tag, err = naming.ParseSecurityTag("snap.pkg_key.hook.configure")
	c.Assert(err, IsNil)
	c.Check(tag.String(), Equals, "snap.pkg_key.hook.configure")
	c.Check(tag.InstanceName(), Equals, "pkg_key")
	c.Check(tag.(naming.HookSecurityTag).HookName(), Equals, "configure")
	c.Check(tag.(naming.HookSecurityTag).ComponentName(), Equals, "")

	tag, err = naming.ParseSecurityTag("snap.pkg+comp.hook.configure")
	c.Assert(err, IsNil)
	c.Check(tag.String(), Equals, "snap.pkg.hook.configure")
	c.Check(tag.InstanceName(), Equals, "pkg")
	c.Check(tag.(naming.HookSecurityTag).HookName(), Equals, "configure")
	c.Check(tag.(naming.HookSecurityTag).ComponentName(), Equals, "comp")

	tag, err = naming.ParseSecurityTag("snap.pkg_key+comp.hook.configure")
	c.Assert(err, IsNil)
	c.Check(tag.String(), Equals, "snap.pkg_key.hook.configure")
	c.Check(tag.InstanceName(), Equals, "pkg_key")
	c.Check(tag.(naming.HookSecurityTag).HookName(), Equals, "configure")
	c.Check(tag.(naming.HookSecurityTag).ComponentName(), Equals, "comp")

	// invalid format is rejected
	_, err = naming.ParseSecurityTag("snap.pkg.app.surprise")
	c.Check(err, ErrorMatches, "invalid security tag")
	_, err = naming.ParseSecurityTag("snap.pkg_key.app.surprise")
	c.Check(err, ErrorMatches, "invalid security tag")

	// invalid snap and app names are rejected.
	_, err = naming.ParseSecurityTag("snap._.app")
	c.Check(err, ErrorMatches, "invalid security tag")
	_, err = naming.ParseSecurityTag("snap.pkg._")
	c.Check(err, ErrorMatches, "invalid security tag")

	// invalid number of components are rejected.
	_, err = naming.ParseSecurityTag("snap.pkg.hook.surprise.")
	c.Check(err, ErrorMatches, "invalid security tag")
	_, err = naming.ParseSecurityTag("snap.pkg.hook.")
	c.Check(err, ErrorMatches, "invalid security tag")
	tag, err = naming.ParseSecurityTag("snap.pkg.hook")
	c.Assert(err, IsNil) // Perhaps somewhat unexpectedly, this tag is valid.
	c.Check(tag.(naming.AppSecurityTag).AppName(), Equals, "hook")
	_, err = naming.ParseSecurityTag("snap.pkg.app.surprise")
	c.Check(err, ErrorMatches, "invalid security tag")
	_, err = naming.ParseSecurityTag("snap.pkg.")
	c.Check(err, ErrorMatches, "invalid security tag")
	_, err = naming.ParseSecurityTag("snap.pkg")
	c.Check(err, ErrorMatches, "invalid security tag")
	_, err = naming.ParseSecurityTag("snap.")
	c.Check(err, ErrorMatches, "invalid security tag")
	_, err = naming.ParseSecurityTag("snap")
	c.Check(err, ErrorMatches, "invalid security tag")
	_, err = naming.ParseSecurityTag("snap.pkg+.hook.install")
	c.Check(err, ErrorMatches, "invalid security tag")
	_, err = naming.ParseSecurityTag("snap.pkg+comp+comp.hook.install")
	c.Check(err, ErrorMatches, "invalid security tag")
	_, err = naming.ParseSecurityTag("snap.pkG+comp.hook.install")
	c.Check(err, ErrorMatches, "invalid security tag")
	_, err = naming.ParseSecurityTag("snap.pkg+comp.app")
	c.Check(err, ErrorMatches, "invalid security tag")

	// things that are not snap.* tags
	_, err = naming.ParseSecurityTag("foo.bar.froz")
	c.Check(err, ErrorMatches, "invalid security tag")
}

func (s *tagSuite) TestParseAppSecurityTag(c *C) {
	// Invalid security tags cannot be parsed.
	tag, err := naming.ParseAppSecurityTag("potato")
	c.Assert(err, ErrorMatches, "invalid security tag")
	c.Assert(tag, IsNil)

	// App security tags can be parsed.
	tag, err = naming.ParseAppSecurityTag("snap.pkg.app")
	c.Assert(err, IsNil)
	c.Check(tag.String(), Equals, "snap.pkg.app")
	c.Check(tag.InstanceName(), Equals, "pkg")
	c.Check(tag.AppName(), Equals, "app")

	// Hook security tags are not app security tags.
	tag, err = naming.ParseAppSecurityTag("snap.pkg.hook.configure")
	c.Assert(err, ErrorMatches, `"snap.pkg.hook.configure" is not an app security tag`)
	c.Assert(tag, IsNil)
}

func (s *tagSuite) TestParseHookSecurityTag(c *C) {
	// Invalid security tags cannot be parsed.
	tag, err := naming.ParseHookSecurityTag("potato")
	c.Assert(err, ErrorMatches, "invalid security tag")
	c.Assert(tag, IsNil)

	// Hook security tags can be parsed.
	tag, err = naming.ParseHookSecurityTag("snap.pkg.hook.configure")
	c.Assert(err, IsNil)
	c.Check(tag.String(), Equals, "snap.pkg.hook.configure")
	c.Check(tag.InstanceName(), Equals, "pkg")
	c.Check(tag.HookName(), Equals, "configure")

	// App security tags are not hook security tags.
	tag, err = naming.ParseHookSecurityTag("snap.pkg.app")
	c.Assert(err, ErrorMatches, `"snap.pkg.app" is not a hook security tag`)
	c.Assert(tag, IsNil)
}

func (s *tagSuite) TestParseWorkloadSecurityTag(c *C) {
	// Invalid security tags cannot be parsed.
	tag, err := naming.ParseWorkloadSecurityTag("potato")
	c.Assert(err, ErrorMatches, "invalid security tag")
	c.Assert(tag, IsNil)

	// Static workload security tags can be parsed.
	tag, err = naming.ParseWorkloadSecurityTag("snap.foo.workload.restricted")
	c.Assert(err, IsNil)
	wtag, ok := tag.(naming.WorkloadSecurityTag)
	c.Assert(ok, Equals, true)
	c.Check(wtag.String(), Equals, "snap.foo.workload.restricted")
	c.Check(wtag.InstanceName(), Equals, "foo")
	c.Check(wtag.WorkloadName(), Equals, "restricted")
	c.Check(wtag.WorkloadInstance(), Equals, "")

	// Workload instance security tags can be parsed (underscore separator).
	tag, err = naming.ParseWorkloadSecurityTag("snap.foo.workload.restricted_inst1")
	c.Assert(err, IsNil)
	wtag = tag.(naming.WorkloadSecurityTag)
	c.Check(wtag.String(), Equals, "snap.foo.workload.restricted_inst1")
	c.Check(wtag.WorkloadName(), Equals, "restricted")
	c.Check(wtag.WorkloadInstance(), Equals, "inst1")

	// Non-workload security tags are not workload security tags.
	tag, err = naming.ParseWorkloadSecurityTag("snap.pkg.app")
	c.Assert(err, ErrorMatches, `"snap.pkg.app" is not a workload security tag`)
	c.Assert(tag, IsNil)

	tag, err = naming.ParseWorkloadSecurityTag("snap.pkg.hook.configure")
	c.Assert(err, ErrorMatches, `"snap.pkg.hook.configure" is not a workload security tag`)
	c.Assert(tag, IsNil)
}

func (s *tagSuite) TestParseWorkloadSecurityTagParallelInstall(c *C) {
	// Parallel install static workload.
	tag, err := naming.ParseWorkloadSecurityTag("snap.foo_bar.workload.container")
	c.Assert(err, IsNil)
	wtag := tag.(naming.WorkloadSecurityTag)
	c.Check(wtag.String(), Equals, "snap.foo_bar.workload.container")
	c.Check(wtag.InstanceName(), Equals, "foo_bar")
	c.Check(wtag.WorkloadName(), Equals, "container")
	c.Check(wtag.WorkloadInstance(), Equals, "")

	// Parallel install workload instance.
	tag, err = naming.ParseWorkloadSecurityTag("snap.foo_bar.workload.container_inst1")
	c.Assert(err, IsNil)
	wtag = tag.(naming.WorkloadSecurityTag)
	c.Check(wtag.String(), Equals, "snap.foo_bar.workload.container_inst1")
	c.Check(wtag.InstanceName(), Equals, "foo_bar")
	c.Check(wtag.WorkloadName(), Equals, "container")
	c.Check(wtag.WorkloadInstance(), Equals, "inst1")
}

func (s *tagSuite) TestParseWorkloadSecurityTagInvalid(c *C) {
	invalidTags := []string{
		// workload name must be valid
		"snap.foo.workload.bad name",
		"snap.foo.workload.-invalid",
		"snap.foo.workload.",
		// workload instance name must be valid
		"snap.foo.workload.-bad_inst1",
		"snap.foo.workload.name.-bad",
		// empty parts
		"snap.foo.workload._inst1",
		"snap.foo.workload.name_",
	}
	for _, tag := range invalidTags {
		_, err := naming.ParseSecurityTag(tag)
		c.Assert(err, NotNil, Commentf("tag %q", tag))
	}
}

func (s *tagSuite) TestParseSecurityTagWorkloadFromParseSecurityTag(c *C) {
	// ParseSecurityTag should also parse workload tags.
	tag, err := naming.ParseSecurityTag("snap.foo.workload.restricted")
	c.Assert(err, IsNil)
	wtag, ok := tag.(naming.WorkloadSecurityTag)
	c.Assert(ok, Equals, true)
	c.Check(wtag.String(), Equals, "snap.foo.workload.restricted")
	c.Check(wtag.WorkloadName(), Equals, "restricted")

	// Workload instance from ParseSecurityTag
	tag, err = naming.ParseSecurityTag("snap.foo.workload.restricted_inst1")
	c.Assert(err, IsNil)
	wtag = tag.(naming.WorkloadSecurityTag)
	c.Check(wtag.String(), Equals, "snap.foo.workload.restricted_inst1")
	c.Check(wtag.WorkloadName(), Equals, "restricted")
	c.Check(wtag.WorkloadInstance(), Equals, "inst1")
}
