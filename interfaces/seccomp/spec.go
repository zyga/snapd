// -*- Mode: Go; indent-tabs-mode: t -*-

/*
 * Copyright (C) 2016-2017 Canonical Ltd
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

package seccomp

import (
	"bytes"
	"sort"
	"strings"

	"github.com/snapcore/snapd/interfaces"
)

// Specification keeps all the seccomp snippets.
type Specification struct {
	// Rules are indexed by security tag.
	rules        map[string][]Rule
	securityTags []string
}

// AddSnippet adds a new seccomp snippet.
func (spec *Specification) AddSnippet(snippet string) {
	rules, err := ParseSnippet(snippet)
	if err != nil {
		panic(err)
	}
	if len(spec.securityTags) == 0 || len(rules) == 0 {
		return
	}
	if spec.rules == nil {
		spec.rules = make(map[string][]Rule)
	}
	for _, tag := range spec.securityTags {
		spec.rules[tag] = append(spec.rules[tag], rules...)
	}
}

// Snippets returns a synthesized version of added snippets.
func (spec *Specification) Snippets() map[string][]string {
	result := make(map[string][]string, len(spec.rules))
	for tag, rules := range spec.rules {
		if result[tag] == nil {
			result[tag] = make([]string, 0, len(rules))
		}
		sort.Sort(bySysCall(rules))
		for _, rule := range rules {
			result[tag] = append(result[tag], strings.TrimSpace(rule.String()))
		}
	}
	return result
}

// SnippetForTag returns a combined snippet for given security tag with individual snippets
// joined with newline character. Empty string is returned for non-existing security tag.
func (spec *Specification) SnippetForTag(tag string) string {
	var buffer bytes.Buffer
	rules := spec.rules[tag]
	sort.Sort(bySysCall(rules))
	for _, rule := range rules {
		// rules automatically contain the trailing newline.
		buffer.WriteString(rule.String())
	}
	return buffer.String()
}

// RulesForTag returns a list of seccomp rules for the given security tag.
func (spec *Specification) RulesForTag(tag string) []Rule {
	rules := spec.rules[tag]
	sort.Sort(bySysCall(rules))
	return rules
}

// SecurityTags returns a list of security tags which have a snippet.
func (spec *Specification) SecurityTags() []string {
	tags := make([]string, 0, len(spec.rules))
	for t := range spec.rules {
		tags = append(tags, t)
	}
	sort.Strings(tags)
	return tags
}

// Implementation of methods required by interfaces.Specification

// AddConnectedPlug records seccomp-specific side-effects of having a connected plug.
func (spec *Specification) AddConnectedPlug(iface interfaces.Interface, plug *interfaces.Plug, plugAttrs map[string]interface{}, slot *interfaces.Slot, slotAttrs map[string]interface{}) error {
	type definer interface {
		SecCompConnectedPlug(spec *Specification, plug *interfaces.Plug, plugAttrs map[string]interface{}, slot *interfaces.Slot, slotAttrs map[string]interface{}) error
	}
	if iface, ok := iface.(definer); ok {
		spec.securityTags = plug.SecurityTags()
		defer func() { spec.securityTags = nil }()
		return iface.SecCompConnectedPlug(spec, plug, plugAttrs, slot, slotAttrs)
	}
	return nil
}

// AddConnectedSlot records seccomp-specific side-effects of having a connected slot.
func (spec *Specification) AddConnectedSlot(iface interfaces.Interface, plug *interfaces.Plug, plugAttrs map[string]interface{}, slot *interfaces.Slot, slotAttrs map[string]interface{}) error {
	type definer interface {
		SecCompConnectedSlot(spec *Specification, plug *interfaces.Plug, plugAttrs map[string]interface{}, slot *interfaces.Slot, slotAttrs map[string]interface{}) error
	}
	if iface, ok := iface.(definer); ok {
		spec.securityTags = slot.SecurityTags()
		defer func() { spec.securityTags = nil }()
		return iface.SecCompConnectedSlot(spec, plug, plugAttrs, slot, slotAttrs)
	}
	return nil
}

// AddPermanentPlug records seccomp-specific side-effects of having a plug.
func (spec *Specification) AddPermanentPlug(iface interfaces.Interface, plug *interfaces.Plug) error {
	type definer interface {
		SecCompPermanentPlug(spec *Specification, plug *interfaces.Plug) error
	}
	if iface, ok := iface.(definer); ok {
		spec.securityTags = plug.SecurityTags()
		defer func() { spec.securityTags = nil }()
		return iface.SecCompPermanentPlug(spec, plug)
	}
	return nil
}

// AddPermanentSlot records seccomp-specific side-effects of having a slot.
func (spec *Specification) AddPermanentSlot(iface interfaces.Interface, slot *interfaces.Slot) error {
	type definer interface {
		SecCompPermanentSlot(spec *Specification, slot *interfaces.Slot) error
	}
	if iface, ok := iface.(definer); ok {
		spec.securityTags = slot.SecurityTags()
		defer func() { spec.securityTags = nil }()
		return iface.SecCompPermanentSlot(spec, slot)
	}
	return nil
}
