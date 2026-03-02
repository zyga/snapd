// -*- Mode: Go; indent-tabs-mode: t -*-

package testutil_test

import (
	"os"

	. "gopkg.in/check.v1"

	"github.com/snapcore/snapd/testutil"
)

var _ = Suite(&envMockSuite{})

type envMockSuite struct{}

func (s *envMockSuite) TestMockEnv(c *C) {
	const envA = "SNAPD_TESTUTIL_MOCKENV_A"
	const envB = "SNAPD_TESTUTIL_MOCKENV_B"
	const envC = "SNAPD_TESTUTIL_MOCKENV_C"

	oldA, hadA := os.LookupEnv(envA)
	oldB, hadB := os.LookupEnv(envB)
	oldC, hadC := os.LookupEnv(envC)
	defer func() {
		if hadA {
			os.Setenv(envA, oldA)
		} else {
			os.Unsetenv(envA)
		}
		if hadB {
			os.Setenv(envB, oldB)
		} else {
			os.Unsetenv(envB)
		}
		if hadC {
			os.Setenv(envC, oldC)
		} else {
			os.Unsetenv(envC)
		}
	}()

	c.Assert(os.Unsetenv(envA), IsNil)
	c.Assert(os.Setenv(envB, "existing"), IsNil)
	c.Assert(os.Unsetenv(envC), IsNil)

	restore := testutil.MockEnv(map[string]string{
		envA: "new-a",
		envB: "",
		envC: "new-c",
	})

	valA, okA := os.LookupEnv(envA)
	c.Check(okA, Equals, true)
	c.Check(valA, Equals, "new-a")

	_, okB := os.LookupEnv(envB)
	c.Check(okB, Equals, false)

	valC, okC := os.LookupEnv(envC)
	c.Check(okC, Equals, true)
	c.Check(valC, Equals, "new-c")

	restore()

	_, okA = os.LookupEnv(envA)
	c.Check(okA, Equals, false)

	valB, okB := os.LookupEnv(envB)
	c.Check(okB, Equals, true)
	c.Check(valB, Equals, "existing")

	_, okC = os.LookupEnv(envC)
	c.Check(okC, Equals, false)
}
