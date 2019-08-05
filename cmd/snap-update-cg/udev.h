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

#pragma once

#include "../libsnap-confine-private/error.h"

typedef struct cgroup_funcs {
    int (*reset)(void *, sc_error **errorp);
    int (*allow)(void *, char device_type, unsigned major, unsigned minor, sc_error **errorp);
} cgroup_funcs;

typedef struct cgroup_iface {
    void *obj;
    const cgroup_funcs *fntab;
} cgroup_iface;

inline int cgroup_iface_reset(cgroup_iface cgif, sc_error **errorp) { return cgif.fntab->reset(cgif.obj, errorp); }

inline int cgroup_iface_allow(cgroup_iface cgif, char device_type, unsigned major, unsigned minor, sc_error **errorp) {
    return cgif.fntab->allow(cgif.obj, device_type, major, minor, errorp);
}

int udev_setup_device_cgroup(const char *udev_tag, cgroup_iface cgif, sc_error **errorp);
