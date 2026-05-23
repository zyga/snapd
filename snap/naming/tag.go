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

package naming

import (
	"errors"
	"fmt"
	"strings"
)

var errInvalidSecurityTag = errors.New("invalid security tag")

// SecurityTag exposes details of a validated snap security tag.
type SecurityTag interface {
	// String returns the entire security tag.
	String() string

	// InstanceName returns the snap name and instance key.
	InstanceName() string
}

// AppSecurityTag exposes details of a validated snap application security tag.
type AppSecurityTag interface {
	SecurityTag
	// AppName returns the name of the application.
	AppName() string
}

type appSecurityTag struct {
	instanceName string
	appName      string
}

func (t appSecurityTag) String() string {
	return fmt.Sprintf("snap.%s.%s", t.instanceName, t.appName)
}

func (t appSecurityTag) InstanceName() string {
	return t.instanceName
}

func (t appSecurityTag) AppName() string {
	return t.appName
}

// HookSecurityTag exposes details of a validated snap hook security tag.
type HookSecurityTag interface {
	SecurityTag
	// HookName returns the name of the hook.
	HookName() string

	// ComponentName returns the name of the component that this hook is
	// associated with, if there is one. If this hook isn't a component hook,
	// this will return an empty string.
	ComponentName() string
}

type hookSecurityTag struct {
	snapName      string
	instanceKey   string
	hookName      string
	componentName string
}

func (t hookSecurityTag) String() string {
	snapInstance := t.snapName
	if t.instanceKey != "" {
		snapInstance = t.snapName + "_" + t.instanceKey
	}
	return fmt.Sprintf("snap.%s.hook.%s", snapInstance, t.hookName)
}

func (t hookSecurityTag) InstanceName() string {
	snapInstance := t.snapName
	if t.instanceKey != "" {
		snapInstance = t.snapName + "_" + t.instanceKey
	}
	return snapInstance
}

func (t hookSecurityTag) HookName() string {
	return t.hookName
}

func (t hookSecurityTag) ComponentName() string {
	return t.componentName
}

// WorkloadSecurityTag exposes details of a validated workload security tag.
type WorkloadSecurityTag interface {
	SecurityTag
	// WorkloadName returns the name of the workload.
	WorkloadName() string
	// WorkloadInstance returns the workload instance name, if any. Empty string
	// for static workloads, non-empty for workload instances.
	WorkloadInstance() string
}

type workloadSecurityTag struct {
	snapName         string
	instanceKey      string
	workloadName     string
	workloadInstance string
}

func (t workloadSecurityTag) String() string {
	snapInstance := t.snapName
	if t.instanceKey != "" {
		snapInstance = t.snapName + "_" + t.instanceKey
	}
	if t.workloadInstance != "" {
		return fmt.Sprintf("snap.%s.workload.%s_%s", snapInstance, t.workloadName, t.workloadInstance)
	}
	return fmt.Sprintf("snap.%s.workload.%s", snapInstance, t.workloadName)
}

func (t workloadSecurityTag) InstanceName() string {
	snapInstance := t.snapName
	if t.instanceKey != "" {
		snapInstance = t.snapName + "_" + t.instanceKey
	}
	return snapInstance
}

func (t workloadSecurityTag) WorkloadName() string {
	return t.workloadName
}

func (t workloadSecurityTag) WorkloadInstance() string {
	return t.workloadInstance
}

// ParseSecurityTag parses a snap security tag and returns a parsed representation or an error.
//
// Further type assertions can be used to described the particular form, either
// describing an application, a hook, or a workload specific security tag.
func ParseSecurityTag(tag string) (SecurityTag, error) {
	// We expect at most five parts (3 for app, 4 for hook/workload, 5 for
	// workload instance). Split with up to six parts so that the len(parts)
	// test catches invalid format tags very early.
	parts := strings.SplitN(tag, ".", 6)
	// We expect three, four, or five components.
	if len(parts) != 3 && len(parts) != 4 && len(parts) != 5 {
		return nil, errInvalidSecurityTag
	}
	// We expect "snap" and the snap instance name as first two fields.
	snapLiteral, instanceName := parts[0], parts[1]
	if snapLiteral != "snap" {
		return nil, errInvalidSecurityTag
	}

	// Split instance name to extract snap name and optional component.
	// For component hooks the format is snap.$snap+component.hook.$hook,
	// where the component name appears after the "+".
	snapName, componentName, hasComponent := strings.Cut(instanceName, "+")
	if err := ValidateInstance(snapName); err != nil {
		return nil, errInvalidSecurityTag
	}
	if hasComponent {
		// Component names must be non-empty and valid (same as app names).
		if componentName == "" || !ValidApp.MatchString(componentName) {
			return nil, errInvalidSecurityTag
		}
	}

	// Three-part tags: snap.$instance.$app
	if len(parts) == 3 {
		if hasComponent {
			return nil, errInvalidSecurityTag
		}

		appName := parts[2]
		if err := ValidateApp(appName); err != nil {
			return nil, errInvalidSecurityTag
		}
		return &appSecurityTag{instanceName: instanceName, appName: appName}, nil
	}

	// Four-part tags: could be hook or workload
	if len(parts) == 4 {
		second := parts[2]
		third := parts[3]

		if second == "hook" {
			if err := ValidateHook(third); err != nil {
				return nil, errInvalidSecurityTag
			}
			// For component hooks, snapName doesn't have instance key (component uses "+").
			// For non-component hooks, instanceName may have instance key (parallel installs use "_").
			var hSnapName, hInstanceKey string
			if hasComponent {
				hSnapName = snapName
			} else {
				hSnapName, hInstanceKey, _ = strings.Cut(instanceName, "_")
				if hInstanceKey != "" {
					if err := ValidateInstance(hSnapName); err != nil {
						return nil, errInvalidSecurityTag
					}
					if !validInstanceKey.MatchString(hInstanceKey) {
						return nil, errInvalidSecurityTag
					}
				} else {
					hSnapName = instanceName
				}
			}
			return &hookSecurityTag{
				snapName:      hSnapName,
				instanceKey:   hInstanceKey,
				hookName:      third,
				componentName: componentName,
			}, nil
		}

		if second == "workload" {
			// third is either "workloadName" (static) or "workloadName_instance" (instance).
			// Workload names cannot contain "_" (per ValidWorkload), so split on first "_" is unambiguous.
			workloadName, workloadInstance, hasInstance := strings.Cut(third, "_")
			if hasInstance && (workloadName == "" || workloadInstance == "") {
				return nil, errInvalidSecurityTag
			}
			if err := ValidateWorkload(workloadName); err != nil {
				return nil, errInvalidSecurityTag
			}
			if hasInstance {
				if err := ValidateWorkload(workloadInstance); err != nil {
					return nil, errInvalidSecurityTag
				}
			}
			// Split instance name to get snapName and instanceKey (parallel installs use "_").
			wSnapName, wInstanceKey, hasInstanceKey := strings.Cut(snapName, "_")
			if hasInstanceKey {
				if err := ValidateInstance(wSnapName); err != nil {
					return nil, errInvalidSecurityTag
				}
				if !validInstanceKey.MatchString(wInstanceKey) {
					return nil, errInvalidSecurityTag
				}
			} else {
				wSnapName = snapName
			}
			return &workloadSecurityTag{
				snapName:         wSnapName,
				instanceKey:      wInstanceKey,
				workloadName:     workloadName,
				workloadInstance: workloadInstance,
			}, nil
		}

		return nil, errInvalidSecurityTag
	}

	// Five-part tags: not used for workloads (workloads use underscore in parts[3] and are 4-dot-part tags).
	// Reject workload tags here since they should have been handled in the 4-part case.
	if len(parts) == 5 && parts[2] == "workload" {
		return nil, errInvalidSecurityTag
	}

	return nil, errInvalidSecurityTag
}

// ParseAppSecurityTag parses an app security tag.
func ParseAppSecurityTag(tag string) (AppSecurityTag, error) {
	parsedTag, err := ParseSecurityTag(tag)
	if err != nil {
		return nil, err
	}
	if parsedAppTag, ok := parsedTag.(AppSecurityTag); ok {
		return parsedAppTag, nil
	}
	return nil, fmt.Errorf("%q is not an app security tag", tag)
}

// ParseHookSecurityTag parses a hook security tag.
func ParseHookSecurityTag(tag string) (HookSecurityTag, error) {
	parsedTag, err := ParseSecurityTag(tag)
	if err != nil {
		return nil, err
	}
	if parsedHookTag, ok := parsedTag.(HookSecurityTag); ok {
		return parsedHookTag, nil
	}
	return nil, fmt.Errorf("%q is not a hook security tag", tag)
}

// ParseWorkloadSecurityTag parses a workload security tag.
func ParseWorkloadSecurityTag(tag string) (WorkloadSecurityTag, error) {
	parsedTag, err := ParseSecurityTag(tag)
	if err != nil {
		return nil, err
	}
	if parsedWorkloadTag, ok := parsedTag.(WorkloadSecurityTag); ok {
		return parsedWorkloadTag, nil
	}
	return nil, fmt.Errorf("%q is not a workload security tag", tag)
}
