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

package policy_test

import (
	"fmt"
	"strings"

	. "gopkg.in/check.v1"

	"github.com/snapcore/snapd/asserts"
	"github.com/snapcore/snapd/interfaces/policy"
	"github.com/snapcore/snapd/snap"
	"github.com/snapcore/snapd/snap/snaptest"
)

type baseDeclSuite struct {
	baseDecl *asserts.BaseDeclaration
}

var _ = Suite(&baseDeclSuite{})

func (s *baseDeclSuite) SetUpSuite(c *C) {
	s.baseDecl = asserts.BuiltinBaseDeclaration()
}

func (s *baseDeclSuite) connectCand(c *C, iface, slotYaml, plugYaml string) *policy.ConnectCandidate {
	if slotYaml == "" {
		slotYaml = fmt.Sprintf(`name: slot-snap
slots:
  %s:
`, iface)
	}
	if plugYaml == "" {
		plugYaml = fmt.Sprintf(`name: plug-snap
plugs:
  %s:
`, iface)
	}
	slotSnap := snaptest.MockInfo(c, slotYaml, nil)
	plugSnap := snaptest.MockInfo(c, plugYaml, nil)
	return &policy.ConnectCandidate{
		Plug:            plugSnap.Plugs[iface],
		Slot:            slotSnap.Slots[iface],
		BaseDeclaration: s.baseDecl,
	}
}

func (s *baseDeclSuite) installSlotCand(c *C, iface string, snapType snap.Type, yaml string) *policy.InstallCandidate {
	if yaml == "" {
		yaml = fmt.Sprintf(`name: install-slot-snap
type: %s
slots:
  %s:
`, snapType, iface)
	}
	snap := snaptest.MockInfo(c, yaml, nil)
	return &policy.InstallCandidate{
		Snap:            snap,
		BaseDeclaration: s.baseDecl,
	}
}

func (s *baseDeclSuite) installPlugCand(c *C, iface string, snapType snap.Type, yaml string) *policy.InstallCandidate {
	if yaml == "" {
		yaml = fmt.Sprintf(`name: install-plug-snap
type: %s
plugs:
  %s:
`, snapType, iface)
	}
	snap := snaptest.MockInfo(c, yaml, nil)
	return &policy.InstallCandidate{
		Snap:            snap,
		BaseDeclaration: s.baseDecl,
	}
}

const declTempl = `type: snap-declaration
authority-id: canonical
series: 16
snap-name: @name@
snap-id: @snapid@
publisher-id: @publisher@
@plugsSlots@
timestamp: 2016-09-30T12:00:00Z
sign-key-sha3-384: Jv8_JiHiIzJVcO9M55pPdqSDWUvuhfDIBJUS-3VW7F_idjix7Ffn5qMxB21ZQuij

AXNpZw==`

func (s *baseDeclSuite) mockSnapDecl(c *C, name, snapID, publisher string, plugsSlots string) *asserts.SnapDeclaration {
	encoded := strings.Replace(declTempl, "@name@", name, 1)
	encoded = strings.Replace(encoded, "@snapid@", snapID, 1)
	encoded = strings.Replace(encoded, "@publisher@", publisher, 1)
	if plugsSlots != "" {
		encoded = strings.Replace(encoded, "@plugsSlots@", strings.TrimSpace(plugsSlots), 1)
	} else {
		encoded = strings.Replace(encoded, "@plugsSlots@\n", "", 1)
	}
	a, err := asserts.Decode([]byte(encoded))
	c.Assert(err, IsNil)
	return a.(*asserts.SnapDeclaration)
}

func (s *baseDeclSuite) TestConnectionContent(c *C) {
	// we let connect explicitly as long as content matches (or is absent on both sides)

	// random (Sanitize* will now also block this)
	cand := s.connectCand(c, "content", "", "")
	err := cand.Check()
	c.Check(err, NotNil)

	slotDecl1 := s.mockSnapDecl(c, "slot-snap", "slot-snap-id", "pub1", "")
	plugDecl1 := s.mockSnapDecl(c, "plug-snap", "plug-snap-id", "pub1", "")
	plugDecl2 := s.mockSnapDecl(c, "plug-snap", "plug-snap-id", "pub2", "")

	// same publisher, same content
	cand = s.connectCand(c, "stuff", `name: slot-snap
slots:
  stuff:
    interface: content
    content: mk1
`, `
name: plug-snap
plugs:
  stuff:
    interface: content
    content: mk1
`)
	cand.SlotSnapDeclaration = slotDecl1
	cand.PlugSnapDeclaration = plugDecl1
	err = cand.Check()
	c.Check(err, IsNil)

	// different publisher, same content
	cand.SlotSnapDeclaration = slotDecl1
	cand.PlugSnapDeclaration = plugDecl2
	err = cand.Check()
	c.Check(err, IsNil)

	// same publisher, different content
	cand = s.connectCand(c, "stuff", `
name: slot-snap
slots:
  stuff:
    interface: content
    content: mk1
`, `
name: plug-snap
plugs:
  stuff:
    interface: content
    content: mk2
`)
	cand.SlotSnapDeclaration = slotDecl1
	cand.PlugSnapDeclaration = plugDecl1
	err = cand.Check()
	c.Check(err, NotNil)
}
