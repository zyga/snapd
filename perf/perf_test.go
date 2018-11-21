// -*- Mode: Go; indent-tabs-mode: t -*-

/*
 * Copyright (C) 2018 Canonical Ltd
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

package perf_test

import (
	"testing"

	. "gopkg.in/check.v1"

	"github.com/snapcore/snapd/perf"
)

func TestPerf(t *testing.T) { TestingT(t) }

type perfSuite struct{}

var _ = Suite(&perfSuite{})

func (*perfSuite) SetUpTest(c *C) {
	perf.ResetRingBuffer()
	perf.ResetNextID()
	c.Assert(perf.CurrentID(), Equals, uint64(0))
}

// NextID returns consecutive integers.
func (*perfSuite) TestNextID(c *C) {
	c.Check(perf.NextID(), Equals, uint64(1))
	c.Check(perf.NextID(), Equals, uint64(2))
	c.Check(perf.NextID(), Equals, uint64(3))
}

// Measure assigns timing and ID.
func (*perfSuite) TestMeasure(c *C) {
	sample := perf.Measure(func() {}, &perf.Sample{Name: "foo"})
	c.Check(sample.Name, Equals, "foo")
	c.Check(sample.ID, Equals, uint64(1))
	c.Check(sample.StartTime.IsZero(), Equals, false)
	c.Check(sample.EndTime.IsZero(), Equals, false)
}

// StoreSample stores a sample into the buffer.
func (*perfSuite) TestStoreSample(c *C) {
	perf.StoreSample(&perf.Sample{ID: 1})
	buf := perf.GetRingBuffer()
	c.Assert(buf, NotNil)
	c.Check(buf.Samples(), DeepEquals, []perf.Sample{{ID: 1}})
}

// The ring buffer in the state cache can be replaced.
func (*perfSuite) TestReplaceRingBuffer(c *C) {
	buf := perf.NewRingBuffer(10)
	perf.ReplaceRingBuffer(buf)
	c.Check(perf.GetRingBuffer(), Equals, buf)
}

// GetRingBuffer returns current ring buffer.
func (*perfSuite) TestGetRingBuffer(c *C) {
	buf := perf.NewRingBuffer(10)
	perf.ReplaceRingBuffer(buf)
	c.Check(perf.GetRingBuffer(), Equals, buf)
}
