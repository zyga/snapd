// -*- Mode: Go; indent-tabs-mode: t -*-

/*
 * Copyright (C) 2019 Canonical Ltd
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
	"github.com/snapcore/snapd/interfaces/apparmor"
	"github.com/snapcore/snapd/interfaces/builtin"
	"github.com/snapcore/snapd/interfaces/hotplug"
	"github.com/snapcore/snapd/interfaces/udev"
	"github.com/snapcore/snapd/snap"
	"github.com/snapcore/snapd/snap/snaptest"
	"github.com/snapcore/snapd/testutil"
)

type uioInterfaceSuite struct {
	testutil.BaseTest
	iface          interfaces.Interface
	slotGadgetInfo *snap.SlotInfo
	slotGadget     *interfaces.ConnectedSlot
	plugInfo       *snap.PlugInfo
	plug           *interfaces.ConnectedPlug
}

var _ = Suite(&uioInterfaceSuite{
	iface: builtin.MustInterface("uio"),
})

func (s *uioInterfaceSuite) SetUpTest(c *C) {
	info := snaptest.MockInfo(c, `
name: gadget
version: 0
type: gadget
slots:
  uio-0:
    interface: uio
    path: /dev/uio0
`, nil)
	s.slotGadgetInfo = info.Slots["uio-0"]
	s.slotGadget = interfaces.NewConnectedSlot(s.slotGadgetInfo, nil, nil)

	info = snaptest.MockInfo(c, `
name: consumer
version: 0
plugs:
  uio:
    interface: uio
apps:
  app:
    command: foo
`, nil)
	s.plugInfo = info.Plugs["uio"]
	s.plug = interfaces.NewConnectedPlug(s.plugInfo, nil, nil)
}

func (s *uioInterfaceSuite) TestName(c *C) {
	c.Assert(s.iface.Name(), Equals, "uio")
}

func (s *uioInterfaceSuite) TestSanitizeSlot(c *C) {
	c.Assert(interfaces.BeforePrepareSlot(s.iface, s.slotGadgetInfo), IsNil)
}

func (s *uioInterfaceSuite) TestUDevSpec(c *C) {
	spec := &udev.Specification{}
	c.Assert(spec.AddConnectedPlug(s.iface, s.plug, s.slotGadget), IsNil)
	c.Assert(spec.Snippets(), HasLen, 2)
	c.Assert(spec.Snippets(), testutil.Contains, `# uio
KERNEL=="uio0", TAG+="snap_consumer_app"`)
	c.Assert(spec.Snippets(), testutil.Contains, `TAG=="snap_consumer_app", RUN+="/usr/lib/snapd/snap-device-helper $env{ACTION} snap_consumer_app $devpath $major:$minor"`)
}

func (s *uioInterfaceSuite) TestAppArmorSpec(c *C) {
	spec := &apparmor.Specification{}
	c.Assert(spec.AddConnectedPlug(s.iface, s.plug, s.slotGadget), IsNil)
	c.Assert(spec.SecurityTags(), DeepEquals, []string{"snap.consumer.app"})
	c.Assert(spec.SnippetForTag("snap.consumer.app"), Equals, ""+
		"/dev/uio0 rwm,\n"+
		"/sys/devices/platform/**/uio/uio0/** rw,")
}

func (s *uioInterfaceSuite) TestStaticInfo(c *C) {
	si := interfaces.StaticInfoOf(s.iface)
	c.Assert(si.ImplicitOnCore, Equals, false)
	c.Assert(si.ImplicitOnClassic, Equals, false)
	c.Assert(si.Summary, Equals, "allows access to specific uio device")
	c.Assert(si.BaseDeclarationSlots, testutil.Contains, "uio")
}

func (s *uioInterfaceSuite) TestAutoConnect(c *C) {
	c.Check(s.iface.AutoConnect(nil, nil), Equals, true)
}

func (s *uioInterfaceSuite) TestHotplugDeviceDetected(c *C) {
	hotplugIface := s.iface.(hotplug.Definer)

	// Events from the "uio" subsystem define new uio slots.
	di, err := hotplug.NewHotplugDeviceInfo(map[string]string{"DEVPATH": "/devices/platform/stuff/uio/uio0", "DEVNAME": "/dev/uio0", "ACTION": "add", "SUBSYSTEM": "uio"})
	c.Assert(err, IsNil)
	proposedSlot, err := hotplugIface.HotplugDeviceDetected(di)
	c.Assert(err, IsNil)
	c.Assert(proposedSlot, DeepEquals, &hotplug.ProposedSlot{
		Name:  "uio0",
		Attrs: map[string]interface{}{"path": "/dev/uio0"}})

	// Events from other subsystems do not.
	di, err = hotplug.NewHotplugDeviceInfo(map[string]string{"DEVPATH": "/devices/platform/stuff/foo/foo0", "DEVNAME": "/dev/foo0", "ACTION": "add", "SUBSYSTEM": "foo"})
	c.Assert(err, IsNil)
	proposedSlot, err = hotplugIface.HotplugDeviceDetected(di)
	c.Assert(err, IsNil)
	c.Assert(proposedSlot, IsNil)
}

func (s *uioInterfaceSuite) TestHotplugKey(c *C) {
	keyHandlerIface := s.iface.(hotplug.HotplugKeyHandler)

	di, err := hotplug.NewHotplugDeviceInfo(map[string]string{"DEVPATH": "/devices/platform/stuff/uio/uio0", "DEVNAME": "/dev/uio0", "ACTION": "add", "SUBSYSTEM": "uio"})
	c.Assert(err, IsNil)
	key, err := keyHandlerIface.HotplugKey(di)
	c.Assert(err, IsNil)
	c.Assert(key, DeepEquals, snap.HotplugKey("uio:7920f79e2e547416cd88d4184af1c514578890ec920e307ac992657084f1e48f"))

	di, err = hotplug.NewHotplugDeviceInfo(map[string]string{"DEVPATH": "/devices/platform/", "DEVNAME": "/dev/uio0", "ACTION": "add", "SUBSYSTEM": "uio"})
	c.Assert(err, IsNil)
	_, err = keyHandlerIface.HotplugKey(di)
	c.Assert(err, ErrorMatches, `unexpected device path for UIO device: ".+"`)
}

func (s *uioInterfaceSuite) TestHotplugHandledByGadget(c *C) {
	byGadgetPred := s.iface.(hotplug.HandledByGadgetPredicate)
	// Gadget defines uio-0 that corresponds to /dev/uio0 so this hotplug device is handled by gadget.
	di, err := hotplug.NewHotplugDeviceInfo(map[string]string{"DEVPATH": "/devices/platform/stuff/uio/uio0", "DEVNAME": "/dev/uio0", "ACTION": "add", "SUBSYSTEM": "uio"})
	c.Assert(err, IsNil)
	c.Assert(byGadgetPred.HandledByGadget(di, s.slotGadgetInfo), Equals, true)

	// This hotplug event is not handled by the gadget.
	di, err = hotplug.NewHotplugDeviceInfo(map[string]string{"DEVPATH": "/devices/platform/stuff/uio/uio1", "DEVNAME": "/dev/uio1", "ACTION": "add", "SUBSYSTEM": "uio"})
	c.Assert(err, IsNil)
	c.Assert(byGadgetPred.HandledByGadget(di, s.slotGadgetInfo), Equals, false)
}

func (s *uioInterfaceSuite) TestInterfaces(c *C) {
	c.Check(builtin.Interfaces(), testutil.DeepContains, s.iface)
}
