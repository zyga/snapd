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

package snapdenv_test

import (
	"testing"

	. "gopkg.in/check.v1"

	"github.com/snapcore/snapd/snapdenv"
	"github.com/snapcore/snapd/testutil"
)

func Test(t *testing.T) { TestingT(t) }

type snapdenvSuite struct{}

var _ = Suite(&snapdenvSuite{})

func (s *snapdenvSuite) TestTesting(c *C) {
	defer testutil.MockEnv(map[string]string{"SNAPPY_TESTING": ""})()

	{
		restore := testutil.MockEnv(map[string]string{"SNAPPY_TESTING": "1"})
		c.Check(snapdenv.Testing(), Equals, true)
		restore()
	}

	{
		restore := testutil.MockEnv(map[string]string{"SNAPPY_TESTING": ""})
		c.Check(snapdenv.Testing(), Equals, false)
		restore()
	}
}

func (s *snapdenvSuite) TestMockTesting(c *C) {
	defer testutil.MockEnv(map[string]string{"SNAPPY_TESTING": ""})()

	r := snapdenv.MockTesting(true)
	defer r()

	c.Check(snapdenv.Testing(), Equals, true)

	snapdenv.MockTesting(false)
	c.Check(snapdenv.Testing(), Equals, false)
}

func (s *snapdenvSuite) TestUseStagingStore(c *C) {
	defer testutil.MockEnv(map[string]string{"SNAPPY_USE_STAGING_STORE": ""})()

	{
		restore := testutil.MockEnv(map[string]string{"SNAPPY_USE_STAGING_STORE": "1"})
		c.Check(snapdenv.UseStagingStore(), Equals, true)
		restore()
	}

	{
		restore := testutil.MockEnv(map[string]string{"SNAPPY_USE_STAGING_STORE": ""})
		c.Check(snapdenv.UseStagingStore(), Equals, false)
		restore()
	}
}

func (s *snapdenvSuite) TestMockUseStagingStore(c *C) {
	defer testutil.MockEnv(map[string]string{"SNAPPY_USE_STAGING_STORE": ""})()

	r := snapdenv.MockUseStagingStore(true)
	defer r()

	c.Check(snapdenv.UseStagingStore(), Equals, true)

	snapdenv.MockUseStagingStore(false)
	c.Check(snapdenv.UseStagingStore(), Equals, false)
}

func (s *snapdenvSuite) TestPreseeding(c *C) {
	defer testutil.MockEnv(map[string]string{"SNAPD_PRESEED": ""})()

	{
		restore := testutil.MockEnv(map[string]string{"SNAPD_PRESEED": "1"})
		c.Check(snapdenv.Preseeding(), Equals, true)
		restore()
	}

	{
		restore := testutil.MockEnv(map[string]string{"SNAPD_PRESEED": ""})
		c.Check(snapdenv.Preseeding(), Equals, false)
		restore()
	}
}

func (s *snapdenvSuite) TestMockPreseeding(c *C) {
	defer testutil.MockEnv(map[string]string{"SNAPD_PRESEED": ""})()

	r := snapdenv.MockPreseeding(true)
	defer r()

	c.Check(snapdenv.Preseeding(), Equals, true)

	snapdenv.MockPreseeding(false)
	c.Check(snapdenv.Preseeding(), Equals, false)
}
