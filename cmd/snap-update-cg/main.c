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

#include <stdio.h>

#include "../libsnap-confine-private/cleanup-funcs.h"
#include "../libsnap-confine-private/utils.h"

#include "cgroup-v1.h"
#include "prog-args.h"
#include "tag.h"
#include "udev.h"

static const cgroup_funcs cg1_funcs = {
    .reset = (int (*)(void *, sc_error **))cgroup_v1_reset,
    .allow = (int (*)(void *, char, unsigned, unsigned, sc_error **))cgroup_v1_allow,
};

/* TODO: get cgroup path explicitly */
static int update_device_cgroup(const char *cgroup_name, const char *security_tag, sc_error **errorp) {
    sc_error *err = NULL;
    cgroup_v1 SC_CLEANUP(cgroup_v1_cleanup) cg1 = CGROUP_V1_INITIALIZER;
    char *udev_tag SC_CLEANUP(sc_cleanup_string) = NULL;

    /* TODO: detect and support both cgroup v1 and v2. */
    if (cgroup_v1_open(&cg1, cgroup_name, &err) < 0) {
        goto out;
    }
    cgroup_iface cgif = {&cg1, &cg1_funcs};
    /* Derive the udev tag from the snap security tag. Because udev does not
     * allow for dots in tag names, those are replaced by underscores in snapd.
     * We just match that behavior. */
    udev_tag = snap_security_tag_to_udev_tag(security_tag);
    if (udev_setup_device_cgroup(udev_tag, cgif, &err) < 0) {
        goto out;
    }
out:
    return sc_error_forward(errorp, err);
}

int main(int argc, char **argv) {
    sc_error *err = NULL;

    /* Use dedicated scope to release prog_args */
    {
        prog_args SC_CLEANUP(prog_args_cleanup) args = PROG_ARGS_INITIALIZER;
        if (prog_args_parse(&args, &argc, &argv, &err) < 0) {
            goto out;
        }
        if (args.is_version_query) {
            printf("snap-update-device-cgroup %s\n", PACKAGE_VERSION);
            goto out;
        }
        if (update_device_cgroup(args.cgroup_name, args.security_tag, &err) < 0) {
            goto out;
        }
    }
out:
    if (sc_error_match(err, CGROUP_V1_DOMAIN, CGROUP_V1_ENOCGROUP)) {
        printf("cgroup v1 unavailable, ignoring\n");
        sc_error_free(err);
    } else if (sc_error_match(err, CGROUP_V1_DOMAIN, CGROUP_V1_ENODEVICES)) {
        printf("cgroup v1 device controller unavailable, ignoring\n");
        sc_error_free(err);
    } else {
        sc_die_on_error(err);
    }
    return 0;
}
