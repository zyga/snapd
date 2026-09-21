/*
 * Copyright (C) 2021 Canonical Ltd
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

#ifndef SNAP_DEVICE_HELPER_H
#define SNAP_DEVICE_HELPER_H

#include "../libsnap-confine-private/bounds-safety.h"

struct sdh_invocation {
    const char *__null_terminated action;
    const char *__null_terminated tagname;
    const char *__null_terminated major;
    const char *__null_terminated minor;
    const char *__null_terminated subsystem;
};

int snap_device_helper_run(const struct sdh_invocation *inv);

#endif /* SNAP_DEVICE_HELPER_H */
