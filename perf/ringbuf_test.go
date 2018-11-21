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
	. "gopkg.in/check.v1"

	"github.com/snapcore/snapd/perf"
)

type ringbufSuite struct{}

var _ = Suite(&ringbufSuite{})

func (*ringbufSuite) TestNewRingBuffer(c *C) {
	buf := perf.NewRingBuffer(10)
	c.Check(buf.Start(), Equals, 0)
	c.Check(buf.Count(), Equals, 0)
	c.Check(len(buf.Data()), Equals, 10)
	c.Check(cap(buf.Data()), Equals, 10)
}

func (*ringbufSuite) TestStoreAndSamples(c *C) {
	buf := perf.NewRingBuffer(3)
	c.Check(buf.Start(), Equals, 0)
	c.Check(buf.Count(), Equals, 0)
	c.Check(len(buf.Data()), Equals, 3)
	c.Check(cap(buf.Data()), Equals, 3)

	c.Check(buf.Samples(), DeepEquals, []perf.Sample{})

	// store "a"
	buf.Store(&perf.Sample{Name: "a"})
	c.Check(buf.Data(), DeepEquals, []perf.Sample{{Name: "a"}, {}, {}})
	c.Check(buf.Samples(), DeepEquals, []perf.Sample{{Name: "a"}})
	c.Check(buf.Start(), Equals, 0)
	c.Check(buf.Count(), Equals, 1)
	c.Check(len(buf.Data()), Equals, 3)
	c.Check(cap(buf.Data()), Equals, 3)

	// store "b"
	buf.Store(&perf.Sample{Name: "b"})
	c.Check(buf.Data(), DeepEquals, []perf.Sample{{Name: "a"}, {Name: "b"}, {}})
	c.Check(buf.Samples(), DeepEquals, []perf.Sample{{Name: "a"}, {Name: "b"}})
	c.Check(buf.Start(), Equals, 0)
	c.Check(buf.Count(), Equals, 2)
	c.Check(len(buf.Data()), Equals, 3)
	c.Check(cap(buf.Data()), Equals, 3)

	// store "c"
	buf.Store(&perf.Sample{Name: "c"})
	c.Check(buf.Data(), DeepEquals, []perf.Sample{{Name: "a"}, {Name: "b"}, {Name: "c"}})
	c.Check(buf.Samples(), DeepEquals, []perf.Sample{{Name: "a"}, {Name: "b"}, {Name: "c"}})
	c.Check(buf.Start(), Equals, 0)
	c.Check(buf.Count(), Equals, 3)
	c.Check(len(buf.Data()), Equals, 3)
	c.Check(cap(buf.Data()), Equals, 3)

	// store "d"
	buf.Store(&perf.Sample{Name: "d"})
	c.Check(buf.Data(), DeepEquals, []perf.Sample{{Name: "d"}, {Name: "b"}, {Name: "c"}})
	c.Check(buf.Samples(), DeepEquals, []perf.Sample{{Name: "b"}, {Name: "c"}, {Name: "d"}})
	c.Check(buf.Start(), Equals, 1)
	c.Check(buf.Count(), Equals, 3)
	c.Check(len(buf.Data()), Equals, 3)
	c.Check(cap(buf.Data()), Equals, 3)

	// store "e"
	buf.Store(&perf.Sample{Name: "e"})
	c.Check(buf.Data(), DeepEquals, []perf.Sample{{Name: "d"}, {Name: "e"}, {Name: "c"}})
	c.Check(buf.Samples(), DeepEquals, []perf.Sample{{Name: "c"}, {Name: "d"}, {Name: "e"}})
	c.Check(buf.Start(), Equals, 2)
	c.Check(buf.Count(), Equals, 3)
	c.Check(len(buf.Data()), Equals, 3)
	c.Check(cap(buf.Data()), Equals, 3)

	// store "f"
	buf.Store(&perf.Sample{Name: "f"})
	c.Check(buf.Data(), DeepEquals, []perf.Sample{{Name: "d"}, {Name: "e"}, {Name: "f"}})
	c.Check(buf.Samples(), DeepEquals, []perf.Sample{{Name: "d"}, {Name: "e"}, {Name: "f"}})
	c.Check(buf.Start(), Equals, 0)
	c.Check(buf.Count(), Equals, 3)
	c.Check(len(buf.Data()), Equals, 3)
	c.Check(cap(buf.Data()), Equals, 3)

	// store "g"
	buf.Store(&perf.Sample{Name: "g"})
	c.Check(buf.Data(), DeepEquals, []perf.Sample{{Name: "g"}, {Name: "e"}, {Name: "f"}})
	c.Check(buf.Samples(), DeepEquals, []perf.Sample{{Name: "e"}, {Name: "f"}, {Name: "g"}})
	c.Check(buf.Start(), Equals, 1)
	c.Check(buf.Count(), Equals, 3)
	c.Check(len(buf.Data()), Equals, 3)
	c.Check(cap(buf.Data()), Equals, 3)
}

func (*ringbufSuite) TestFilter(c *C) {
	buf := perf.NewRingBuffer(100)
	for i := 0; i < 10; i++ {
		buf.Store(&perf.Sample{ID: uint64(i)})
	}
	odd := buf.Filter(func(s *perf.Sample) bool { return s.ID%2 == 1 })
	c.Check(odd, DeepEquals, []perf.Sample{{ID: 1}, {ID: 3}, {ID: 5}, {ID: 7}, {ID: 9}})
}
