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

#include <stdbool.h>

#include "../libsnap-confine-private/error.h"

#define PROG_ARGS_DOMAIN "prog-args"

enum {
    PROG_ARGS_EUSAGE = 1,
};

typedef struct prog_args {
    /* The relative path of the cgroup to use. */
    char *cgroup_name;
    /* The security tag of a snap application or hook. */
    char *security_tag;
    /* Flag indicating that --version was passed on command line. */
    bool is_version_query;
} prog_args;

#define PROG_ARGS_INITIALIZER \
    { NULL, false }

int prog_args_parse(prog_args *args, int *argcp, char ***argvp, sc_error **errorp);
void prog_args_free(prog_args *args);
void prog_args_cleanup(prog_args *ptr);
