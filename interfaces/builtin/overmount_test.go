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

package builtin_test

import (
	. "gopkg.in/check.v1"

	"github.com/snapcore/snapd/interfaces"
	"github.com/snapcore/snapd/interfaces/builtin"
	"github.com/snapcore/snapd/testutil"
)

type overmountSuite struct {
	iface interfaces.Interface
}

var _ = Suite(&overmountSuite{
	iface: builtin.MustInterface("overmount"),
})

func (s *overmountSuite) TestName(c *C) {
	c.Assert(s.iface.Name(), Equals, "overmount")
}

func (s *overmountSuite) TestMetaData(c *C) {
	md := interfaces.IfaceMetaData(s.iface)
	c.Check(md.ImplicitOnCore, Equals, true)
	c.Check(md.ImplicitOnClassic, Equals, true)
	c.Check(md.Description, testutil.Contains, "The overmount interface allows")
}

func (s *overmountSuite) TestInterfaces(c *C) {
	c.Check(builtin.Interfaces(), testutil.DeepContains, s.iface)
}
