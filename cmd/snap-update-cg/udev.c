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

#include "udev.h"

#include <limits.h>
#include <stdlib.h>
#include <string.h>
#include <sys/stat.h>
#include <sys/sysmacros.h>

#include <libudev.h>

#include "../libsnap-confine-private/cleanup-funcs.h"
#include "../libsnap-confine-private/string-utils.h"
#include "../libsnap-confine-private/utils.h"

/**
 * Allow access to common devices.
 *
 * The list includes /dev/{null,zero,full,random,urandom,tty,console,ptmx}.
 **/
static int cgroup_allow_common(cgroup_iface cgif, sc_error **errorp) {
    sc_error *err = NULL;
    debug("allowing access to common devices");
    /* The devices we add here have static number allocation.
     * https://www.kernel.org/doc/html/v4.11/admin-guide/devices.html */
    dev_t allowed[] = {
        makedev(1, 3),  // /dev/null
        makedev(1, 5),  // /dev/zero
        makedev(1, 7),  // /dev/full
        makedev(1, 8),  // /dev/random
        makedev(1, 9),  // /dev/urandom
        makedev(5, 0),  // /dev/tty
        makedev(5, 1),  // /dev/console
        makedev(5, 2),  // /dev/ptmx
    };
    for (size_t i = 0; i < sizeof allowed / sizeof *allowed; ++i) {
        if (cgroup_iface_allow(cgif, 'c', major(allowed[i]), minor(allowed[i]), &err) < 0) {
            goto out;
        }
    }
out:
    return sc_error_forward(errorp, err);
}

/**
 * Allow access to current and future PTY slaves.
 *
 * We unconditionally add them since we use a devpts newinstance. Unix98 PTY
 * slaves major are 136-143.
 *
 * See also:
 * https://github.com/torvalds/linux/blob/master/Documentation/admin-guide/devices.txt
 **/
static int cgroup_allow_pty_slaves(cgroup_iface cgif, sc_error **errorp) {
    sc_error *err = NULL;
    debug("allowing access to current and future PTY slaves");
    for (unsigned pty_major = 136; pty_major <= 143; pty_major++) {
        if (cgroup_iface_allow(cgif, 'c', pty_major, UINT_MAX, &err) < 0) {
            goto out;
        }
    }
out:
    return sc_error_forward(errorp, err);
}

/**
 * Allow access to NVidia devices.
 *
 * NVidia modules are proprietary and therefore aren't in sysfs and can't be
 * udev tagged. For now, just add existing nvidia devices to the cgroup
 * unconditionally (AppArmor will still mediate the access).  We'll want to
 * rethink this if snapd needs to mediate access to other proprietary devices.
 *
 * Device major and minor numbers are described in (though nvidia-uvm currently
 * isn't listed):
 *
 * https://github.com/torvalds/linux/blob/master/Documentation/admin-guide/devices.txt
 **/
static int cgroup_allow_nvidia(cgroup_iface cgif, sc_error **errorp) {
    sc_error *err = NULL;
    struct stat sbuf;
    debug("allowing access to nvidia devices, if present");
    /* Allow access to /dev/nvidia0 through /dev/nvidia254, stopping after the
     * first one that is not present on the system. */
    for (unsigned nv_minor = 0; nv_minor < 255; nv_minor++) {
        char nv_path[15] = {0};
        sc_must_snprintf(nv_path, sizeof(nv_path), "/dev/nvidia%u", nv_minor);
        if (stat(nv_path, &sbuf) != 0) {
            break;
        }
        if (cgroup_iface_allow(cgif, 'c', major(sbuf.st_rdev), minor(sbuf.st_rdev), &err) < 0) {
            goto out;
        }
    }
    if (stat("/dev/nvidiactl", &sbuf) == 0) {
        if (cgroup_iface_allow(cgif, 'c', major(sbuf.st_rdev), minor(sbuf.st_rdev), &err) < 0) {
            goto out;
        }
    }
    if (stat("/dev/nvidia-uvm", &sbuf) == 0) {
        if (cgroup_iface_allow(cgif, 'c', major(sbuf.st_rdev), minor(sbuf.st_rdev), &err) < 0) {
            goto out;
        }
    }
    if (stat("/dev/nvidia-modeset", &sbuf) == 0) {
        if (cgroup_iface_allow(cgif, 'c', major(sbuf.st_rdev), minor(sbuf.st_rdev), &err) < 0) {
            goto out;
        }
    }
out:
    return sc_error_forward(errorp, err);
}

/**
 * Allow access to /dev/uhid.
 *
 * Currently /dev/uhid isn't represented in sysfs, so add it to the device
 * cgroup if it exists and let AppArmor handle the mediation.
 **/
static int cgroup_allow_uhid(cgroup_iface cgif, sc_error **errorp) {
    sc_error *err = NULL;
    struct stat sbuf;
    debug("allowing access to uhid, if present");
    if (stat("/dev/uhid", &sbuf) == 0) {
        if (cgroup_iface_allow(cgif, 'c', major(sbuf.st_rdev), minor(sbuf.st_rdev), &err) < 0) {
            goto out;
        }
    }
out:
    return sc_error_forward(errorp, err);
}

/**
 * Allow access to assigned devices.
 *
 * The snapd udev security backend uses udev rules to tag matching devices with
 * tags corresponding to snap applications. Here we interrogate udev and allow
 * access to all assigned devices.
 **/
static int cgroup_allow_udev_assigned(cgroup_iface cgif, struct udev *udev, struct udev_list_entry *assigned,
                                      sc_error **errorp) {
    sc_error *err = NULL;
    debug("allowing access to devices udev-tagged to the snap security tag");
    for (struct udev_list_entry *entry = assigned; entry != NULL; entry = udev_list_entry_get_next(entry)) {
        const char *path = udev_list_entry_get_name(entry);
        if (path == NULL) {
            err = sc_error_init_simple("cannot get device path from udev enumeration entry");
            goto out;
        }
        struct udev_device *device = udev_device_new_from_syspath(udev, path);
        if (device == NULL) {
            err = sc_error_init_simple("cannot find device from syspath %s", path);
            goto out;
        }
        dev_t devnum = udev_device_get_devnum(device);
        if (cgroup_iface_allow(cgif, strstr(path, "/block/") != NULL ? 'b' : 'c', major(devnum), minor(devnum), &err) <
            0) {
            goto out;
        }

        udev_device_unref(device);
    }
out:
    return sc_error_forward(errorp, err);
}

static void sc_cleanup_udev(struct udev **udev) {
    if (udev != NULL && *udev != NULL) {
        udev_unref(*udev);
        *udev = NULL;
    }
}

static void sc_cleanup_udev_enumerate(struct udev_enumerate **enumerate) {
    if (enumerate != NULL && *enumerate != NULL) {
        udev_enumerate_unref(*enumerate);
        *enumerate = NULL;
    }
}

int udev_setup_device_cgroup(const char *udev_tag, cgroup_iface cgif, sc_error **errorp) {
    sc_error *err = NULL;
    struct udev SC_CLEANUP(sc_cleanup_udev) *udev = NULL;
    struct udev_enumerate SC_CLEANUP(sc_cleanup_udev_enumerate) *enumerate = NULL;
    /* NOTE: udev_list_entry is bound to life-cycle of the used udev_enumerate */
    struct udev_list_entry *assigned;

    /* Use udev APIs to talk to udev-the-daemon to determine the list of
     * "devices" with that tag assigned. The list may be empty, in which case
     * there's no udev tagging in effect and we must refrain from constructing
     * the cgroup as it would interfere with the execution of a program. */

    debug("looking for devices udev-tagged to the snap security tag");
    udev = udev_new();
    if (udev == NULL) {
        err = sc_error_init_simple("cannot connect to udev");
        goto out;
    }
    enumerate = udev_enumerate_new(udev);
    if (enumerate == NULL) {
        err = sc_error_init_simple("cannot create udev device enumeration");
        goto out;
    }
    if (udev_enumerate_add_match_tag(enumerate, udev_tag) != 0) {
        err = sc_error_init_simple("cannot add tag match to udev device enumeration");
        goto out;
    }
    if (udev_enumerate_scan_devices(enumerate) != 0) {
        err = sc_error_init_simple("cannot enumerate udev devices");
        goto out;
    }
    assigned = udev_enumerate_get_list_entry(enumerate);
    if (cgroup_iface_reset(cgif, &err) < 0) {
        goto out;
    }
    if (assigned != NULL) {
        debug("configuring cgroup to allow access to select devices");
        /* There are some devices that udev has tagged to this snap name and
         * snap application or hook. */
        if (cgroup_allow_common(cgif, &err) < 0) {
            goto out;
        }
        if (cgroup_allow_pty_slaves(cgif, &err) < 0) {
            goto out;
        }
        if (cgroup_allow_nvidia(cgif, &err) < 0) {
            goto out;
        }
        if (cgroup_allow_uhid(cgif, &err) < 0) {
            goto out;
        }
        if (cgroup_allow_udev_assigned(cgif, udev, assigned, &err) < 0) {
            goto out;
        }
    } else {
        debug("configuring cgroup to allow access to all devices");
        /* There are no devices that udev has tagged to this snap name and snap
         * application or hook. Up to snapd 2.40 we would neither create a device
         * cgroup nor move the process to it.
         *
         * This turned out to be a mistake, tracked as bug
         * https://bugs.launchpad.net/snapd/+bug/1838937
         *
         * When a device is subsequently tagged, e.g. by connecting a device to
         * the system or by connecting an interface that tags an existing
         * device, then the process is already in the right cgroup and all that
         * needs to be done is to reconfigure the cgroup from unconfined to one
         * that implements a specific confinement.
         *
         * Setup the cgroup to allow access to all devices. */
        if (cgroup_iface_allow(cgif, 'a', UINT_MAX, UINT_MAX, &err) < 0) {
            goto out;
        }
    }
out:
    return sc_error_forward(errorp, err);
}
