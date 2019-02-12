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

package snapstate

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"

	"github.com/snapcore/snapd/dirs"
	"github.com/snapcore/snapd/snap"
)

func parsePid(text string) (int, error) {
	pid, err := strconv.Atoi(text)
	if err == nil && pid <= 0 {
		return 0, fmt.Errorf("cannot parse pid %q", text)
	}
	return pid, err
}

func parsePids(reader io.Reader) ([]int, error) {
	scanner := bufio.NewScanner(reader)
	var pids []int
	for scanner.Scan() {
		s := scanner.Text()
		pid, err := parsePid(s)
		if err != nil {
			return nil, err
		}
		pids = append(pids, pid)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return pids, nil
}

func pidsOfSnap(snapName string) (map[int]bool, error) {
	fname := filepath.Join(dirs.FreezerCgroupDir, "snap."+snapName, "cgroup.procs")
	file, err := os.Open(fname)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer file.Close()

	pids, err := parsePids(bufio.NewReader(file))
	if err != nil {
		return nil, err
	}

	pidSet := make(map[int]bool, len(pids))
	for _, pid := range pids {
		pidSet[pid] = true
	}
	return pidSet, nil
}

func deletePidsOfSecurityTag(pidSet map[int]bool, securityTag string) error {
	fname := filepath.Join(dirs.PidsCgroupDir, securityTag, "cgroup.procs")
	file, err := os.Open(fname)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	defer file.Close()

	pids, err := parsePids(bufio.NewReader(file))
	if err != nil {
		return err
	}

	for _, pid := range pids {
		delete(pidSet, pid)
	}
	return nil
}

func SoftRefreshCheck(info *snap.Info) error {
	snapName := info.SnapName()
	pidSet, err := pidsOfSnap(snapName)
	if err != nil {
		return err
	}
	for _, app := range info.Apps {
		if !app.IsService() {
			continue
		}
		err := deletePidsOfSecurityTag(pidSet, app.SecurityTag())
		if err != nil {
			return err
		}
	}
	if len(pidSet) > 0 {
		pids := make([]int, 0, len(pidSet))
		for pid := range pidSet {
			pids = append(pids, pid)
		}
		sort.Ints(pids)
		return &BusySnapError{pids: pids, snapName: snapName}
	}
	return nil
}

func HardRefreshCheck(snapName string) error {
	pidSet, err := pidsOfSnap(snapName)
	if err != nil {
		return err
	}
	if len(pidSet) > 0 {
		pids := make([]int, 0, len(pidSet))
		for pid := range pidSet {
			pids = append(pids, pid)
		}
		sort.Ints(pids)
		return &BusySnapError{pids: pids, snapName: snapName}
	}
	return nil
}

type BusySnapError struct {
	pids     []int
	snapName string
}

func (err BusySnapError) Error() string {
	return fmt.Sprintf("snap %q has running apps or hooks", err.snapName)
}

func (err BusySnapError) Pids() []int {
	return err.pids
}
