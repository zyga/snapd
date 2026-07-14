// -*- Mode: Go; indent-tabs-mode: t -*-

/*
 * Copyright (C) 2019-2020 Canonical Ltd
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

package snapdtool

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// SnapdVersionFromInfoFile returns the snapd version read from the info file in
// the given dir, as well as any other key/value pairs/flags in the file.
// See ParseInfoFile for more format details.
func SnapdVersionFromInfoFile(dir string) (version string, flags map[string]string, err error) {
	infoPath := filepath.Join(dir, "info")
	f, err := os.Open(infoPath)
	if err != nil {
		return "", nil, fmt.Errorf("cannot open snapd info file %q: %s", infoPath, err)
	}
	defer f.Close()

	return ParseInfoFile(f, fmt.Sprintf("%q", infoPath))
}

// ParseInfoFile parses the "info" file provided via an io.Reader. It returns
// the snapd version read from the info file, as well as any other key/value
// pairs/flags in the file.
// whence is used to construct error messages as "... info file %s".
// The format of the "info" file are lines with "KEY=VALUE" with the typical key
// being just VERSION. The file is produced by mkversion.sh and normally
// installed along snapd binary in /usr/lib/snapd.
// Other typical keys in this file include SNAPD_APPARMOR_REEXEC, which
// indicates whether or not the snapd-apparmor binary installed via the
// traditional linux package of snapd supports re-exec into the version in the
// snapd or core snaps.
func ParseInfoFile(f io.Reader, whence string) (version string, flags map[string]string, err error) {
	flags = map[string]string{}

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "VERSION=") {
			version = strings.TrimPrefix(line, "VERSION=")
		} else {
			keyVal := strings.SplitN(line, "=", 2)
			if len(keyVal) != 2 {
				// potentially malformed line, just skip it
				continue
			}

			flags[keyVal[0]] = keyVal[1]
		}
	}

	if err := scanner.Err(); err != nil {
		return "", nil, fmt.Errorf("error reading snapd info file %s: %v", whence, err)
	}

	if version == "" {
		return "", nil, fmt.Errorf("cannot find version in snapd info file %s", whence)
	}

	return version, flags, nil
}

// InfoFile represents the parsed contents of a snapd "info" file. The info
// file is produced by mkversion.sh and normally installed along the snapd
// binary in /usr/lib/snapd, or packaged inside core/snapd/kernel snaps as
// /usr/lib/snapd/info or /snapd-info respectively.
//
// The version fields mirror snapdtool.UpstreamVersion/DownstreamVersionSuffix:
// FullVersion is the complete version string (from VERSION=), UpstreamVersion
// and DownstreamVersionSuffix are optional decomposed parts that downstream
// packaging can set independently. When only VERSION= is present, UpstreamVersion
// defaults to FullVersion and DownstreamVersionSuffix is empty.
type InfoFile struct {
	FullVersion             string            // full combined version (from VERSION=)
	UpstreamVersion         string            // upstream part of the version (from UPSTREAM_VERSION=)
	DownstreamVersionSuffix string            // distro-specific suffix (from DOWNSTREAM_VERSION_SUFFIX=)
	Flags                   map[string]string // other key/value pairs in the file
}

// ReadInfoFile reads and parses an info file from the given directory, returning
// a composite InfoFile value with decomposed version fields. This is a new API
// that supersedes SnapdVersionFromInfoFile for callers that need access to
// individual version components (upstream vs downstream suffix). For callers
// that only need the combined version string, SnapdVersionFromInfoFile remains
// suitable and avoids introducing an unused struct value.
// See ParseInfoFileForInfo for more format details.
func ReadInfoFile(dir string) (*InfoFile, error) {
	infoPath := filepath.Join(dir, "info")
	f, err := os.Open(infoPath)
	if err != nil {
		return nil, fmt.Errorf("cannot open snapd info file %q: %s", infoPath, err)
	}
	defer f.Close()

	return ParseInfoFileForInfo(f, fmt.Sprintf("%q", infoPath))
}

// ParseInfoFileForInfo parses the "info" file provided via an io.Reader into a
// composite InfoFile value with decomposed version fields. See ReadInfoFile for
// format details and the rationale behind UpstreamVersion defaulting to
// FullVersion when UPSTREAM_VERSION is not present in the file.
func ParseInfoFileForInfo(f io.Reader, whence string) (*InfoFile, error) {
	info := &InfoFile{
		Flags: map[string]string{},
	}

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if key, val, ok := strings.Cut(line, "="); !ok {
			// potentially malformed line, just skip it
			continue
		} else if key == "VERSION" {
			info.FullVersion = val
		} else if key == "UPSTREAM_VERSION" {
			info.UpstreamVersion = val
		} else if key == "DOWNSTREAM_VERSION_SUFFIX" {
			info.DownstreamVersionSuffix = val
		} else {
			info.Flags[key] = val
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading snapd info file %s: %v", whence, err)
	}

	if info.FullVersion == "" {
		return nil, fmt.Errorf("cannot find version in snapd info file %s", whence)
	}

	// Default upstream to the full version if not explicitly set. This allows
	// callers that only deal with the combined version to work correctly with
	// info files generated by mkversion.sh (which writes only VERSION=).
	if info.UpstreamVersion == "" {
		info.UpstreamVersion = info.FullVersion
	}

	return info, nil
}
