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

package inputwatch

import (
	"errors"
	"fmt"
	"time"

	"github.com/snapcore/snapd/logger"
)

type inputProvider interface {
	FindMatchingDevices(filter inputEventFilter) ([]inputDevice, error)
}

type keyEvent struct {
	Dev inputDevice
	Err error
}

type inputDevice interface {
	WaitForTrigger(chan keyEvent)
	String() string
}

type inputEventFilter struct {
	Key string
}

var (
	// input mechanism
	input inputProvider

	// wait for '1' to be pressed
	triggerFilter = inputEventFilter{Key: "KEY_1"}

	// key wait timeout
	timeout = 5 * time.Second

	ErrKeyNotDetected = errors.New("interrupt key not detected")
)

// WaitTriggerKey wait for trigger key on the available input devices. Returns
// nil if one was detected, ErrKeyNotDetected if there was none, or other
// non-nil error.
func WaitTriggerKey() error {
	if input == nil {
		logger.Panicf("input is unset")
	}

	devices, err := input.FindMatchingDevices(triggerFilter)
	if err != nil {
		return fmt.Errorf("cannot list input devices: %v", err)
	}
	if devices == nil {
		return fmt.Errorf("cannot find matching devices")
	}

	logger.Noticef("waiting for key: %v", chooserTriggerKey.Name)

	// wait for a couple of second for the key
	detectKeyCh := make(chan keyEvent, len(devices))

	for _, kbd := range devices {
		go kbd.WaitForTrigger(detectKeyCh)
	}

	select {
	case kev := <-detectKeyCh:
		if kev.Err == nil {
			// channel got closed without an error
			logger.Noticef("%s: + got key %v", kev.Dev, chooserTriggerKey)
		}
		return err
	case <-time.After(timeout):
		logger.Noticef("- no key detected")
		return ErrKeyNotDetected
	}

	return nil
}
