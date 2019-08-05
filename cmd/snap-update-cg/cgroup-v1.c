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

#include "config.h"

#include "cgroup-v1.h"

#include <errno.h>
#include <fcntl.h>
#include <limits.h>
#include <stdlib.h>
#include <sys/stat.h>
#include <sys/types.h>
#include <unistd.h>

#include "../libsnap-confine-private/cleanup-funcs.h"
#include "../libsnap-confine-private/utils.h"

int cgroup_v1_open(cgroup_v1 *cg1, const char *cgroup_name, sc_error **errorp) {
    sc_error *err = NULL;
    const char *base_path = "/sys/fs/cgroup";
    int SC_CLEANUP(sc_cleanup_close) base_fd = -1;
    const char *devices_relpath = "devices";
    int SC_CLEANUP(sc_cleanup_close) devices_fd = -1;
    const char *cgroup_relpath = cgroup_name;
    int SC_CLEANUP(sc_cleanup_close) cgroup_fd = -1;
    const char *devices_allow_relpath = "devices.allow";
    int devices_allow_fd = -1;
    const char *devices_deny_relpath = "devices.deny";
    int devices_deny_fd = -1;

    /* Open /sys/fs/cgroup */
    base_fd = open(base_path, O_PATH | O_DIRECTORY | O_CLOEXEC | O_NOFOLLOW);
    if (base_fd < 0 && errno == ENOENT) {
        if (errno == ENOENT) {
            /* This system does not support cgroups. */
            err = sc_error_init(CGROUP_V1_DOMAIN, CGROUP_V1_ENOCGROUP, "cannot open %s", base_path);
        } else {
            err = sc_error_init_from_errno(errno, "cannot open %s", base_path);
        }
        goto out;
    }

    /* Open devices relative to /sys/fs/cgroup */
    devices_fd = openat(base_fd, devices_relpath, O_PATH | O_DIRECTORY | O_CLOEXEC | O_NOFOLLOW);
    if (devices_fd < 0) {
        if (errno == ENOENT) {
            /* This system does not support the device cgroup. */
            err =
                sc_error_init(CGROUP_V1_DOMAIN, CGROUP_V1_ENODEVICES, "cannot open %s/%s", base_path, devices_relpath);
        } else {
            err = sc_error_init_from_errno(errno, "cannot open %s/%s", base_path, devices_relpath);
        }
        goto out;
    }

    /* Open snap.$SNAP_NAME.$APP_NAME relative to /sys/fs/cgroup/devices,
     * creating the directory if necessary. Note that we always chown the
     * resulting directory to root:root. */
    if (mkdirat(devices_fd, cgroup_relpath, 0755) < 0) {
        if (errno != EEXIST) {
            err = sc_error_init_from_errno(errno, "cannot create directory %s/%s/%s", base_path, devices_relpath,
                                           cgroup_relpath);
            goto out;
        }
    }

    cgroup_fd = openat(devices_fd, cgroup_relpath, O_RDONLY | O_DIRECTORY | O_CLOEXEC | O_NOFOLLOW);
    if (cgroup_fd < 0) {
        err = sc_error_init_from_errno(errno, "cannot open %s/%s/%s", base_path, devices_relpath, cgroup_relpath);
        goto out;
    }
    if (fchown(cgroup_fd, 0, 0) < 0) {
        err = sc_error_init_from_errno(errno, "cannot chown %s/%s/%s to root:root", base_path, devices_relpath,
                                       cgroup_relpath);
        goto out;
    }

    /* Open devices.allow relative to /sys/fs/cgroup/devices/snap.$SNAP_NAME.$APP_NAME */
    devices_allow_fd = openat(cgroup_fd, devices_allow_relpath, O_WRONLY | O_CLOEXEC | O_NOFOLLOW);
    if (devices_allow_fd < 0) {
        err = sc_error_init_from_errno(errno, "cannot open %s/%s/%s/%s", base_path, devices_relpath, cgroup_relpath,
                                       devices_allow_relpath);
        goto out;
    }

    /* Open devices.deny relative to /sys/fs/cgroup/devices/snap.$SNAP_NAME.$APP_NAME */
    devices_deny_fd = openat(cgroup_fd, devices_deny_relpath, O_WRONLY | O_CLOEXEC | O_NOFOLLOW);
    if (devices_deny_fd < 0) {
        err = sc_error_init_from_errno(errno, "cannot open %s/%s/%s/%s", base_path, devices_relpath, cgroup_relpath,
                                       devices_deny_relpath);
        goto out;
    }

    /* Everything worked so pack the result and "move" the descriptors over so
     * that they are not closed by the cleanup functions. */
    cg1->devices_allow_fd = devices_allow_fd;
    cg1->devices_deny_fd = devices_deny_fd;
    devices_allow_fd = -1;
    devices_deny_fd = -1;

out:
    return sc_error_forward(errorp, err);
}

void cgroup_v1_close(cgroup_v1 *cg1) {
    sc_cleanup_close(&cg1->devices_allow_fd);
    sc_cleanup_close(&cg1->devices_deny_fd);
}

void cgroup_v1_cleanup(cgroup_v1 *cg1) {
    if (cg1 != NULL) {
        cgroup_v1_close(cg1);
    }
}

int cgroup_v1_reset(cgroup_v1 *cg1, sc_error **errorp) {
    sc_error *err = NULL;
    /* Write 'a' to devices.deny to remove all existing devices that were
     * added in previous invocations. */
    if (dprintf(cg1->devices_deny_fd, "a") < 0) {
        err = sc_error_init_simple("cannot reset access list");
        goto out;
    }
    debug("reset access list");
out:
    return sc_error_forward(errorp, err);
}

int cgroup_v1_allow(cgroup_v1 *cg1, char device_type, unsigned major, unsigned minor, sc_error **errorp) {
    sc_error *err = NULL;
    if (device_type != 'a' && device_type != 'c' && device_type != 'b') {
        err = sc_error_init_api_misuse("device_type must be one of 'a', 'c' or 'b'");
        goto out;
    }
    const char *device_type_str =
        (device_type == 'c' ? "character"
                            : (device_type == 'b' ? "block" : (device_type == 'a' ? "character and block" : "???")));

    if (major != UINT_MAX && minor != UINT_MAX) {
        if (dprintf(cg1->devices_allow_fd, "%c %u:%u rwm", device_type, major, minor) < 0) {
            err = sc_error_init_simple("cannot allow device access: '%c %u:%u rwm'", device_type, major, minor);
            goto out;
        }
        debug("allow access to %s device with major:minor %u:%u", device_type_str, major, minor);
    } else if (major == UINT_MAX && minor != UINT_MAX) {
        if (dprintf(cg1->devices_allow_fd, "%c *:%u rwm", device_type, minor) < 0) {
            err = sc_error_init_simple("cannot allow device access: '%c *:%u rwm'", device_type, minor);
            goto out;
        }
        debug("allow access to %s device with major:minor (any):%u", device_type_str, minor);
    } else if (major != UINT_MAX && minor == UINT_MAX) {
        if (dprintf(cg1->devices_allow_fd, "%c %u:* rwm", device_type, major) < 0) {
            err = sc_error_init_simple("cannot allow device access: '%c %u:* rwm'", device_type, major);
            goto out;
        }
        debug("allow access to %s device with major:minor %u:(any)", device_type_str, major);
    } else if (major == UINT_MAX && minor == UINT_MAX) {
        if (dprintf(cg1->devices_allow_fd, "%c *:* rwm", device_type) < 0) {
            err = sc_error_init_simple("cannot allow device access: '%c *:* rwm'", device_type);
            goto out;
        }
        debug("allow access to %s device with major:minor (any):(any)", device_type_str);
    }
out:
    return sc_error_forward(errorp, err);
}
