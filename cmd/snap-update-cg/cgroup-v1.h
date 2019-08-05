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

#define CGROUP_V1_DOMAIN "cgroup-v1"

enum {
    CGROUP_V1_ENOCGROUP = 1,
    CGROUP_V1_ENODEVICES = 1,
};

typedef struct cgroup_v1 {
    int devices_allow_fd;
    int devices_deny_fd;
} cgroup_v1;

#define CGROUP_V1_INITIALIZER \
    { -1, -1 }

int cgroup_v1_open(cgroup_v1 *cg1, const char *cgroup_name, sc_error **errorp);
void cgroup_v1_close(cgroup_v1 *cg1);
void cgroup_v1_cleanup(cgroup_v1 *cg1);

int cgroup_v1_reset(cgroup_v1 *cg1, sc_error **errorp);
int cgroup_v1_allow(cgroup_v1 *cg1, char device_type, unsigned major, unsigned minor, sc_error **errorp);
