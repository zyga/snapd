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

#include "prog-args.h"

#include <regex.h>
#include <stdio.h>
#include <string.h>

#include "../libsnap-confine-private/cleanup-funcs.h"
#include "../libsnap-confine-private/string-utils.h"
#include "../libsnap-confine-private/utils.h"

static int check_snap_security_tag(const char *security_tag, sc_error **errorp) {
    const char *whitelist_re =
        "^snap\\.([a-z0-9](-?[a-z0-9])*(_[a-z0-9]{1,10})?)\\.([a-zA-Z0-9](-?[a-zA-Z0-9])*|hook\\.[a-z](-?[a-z])*)$";
    sc_error *err = NULL;

    regex_t re;
    if (regcomp(&re, whitelist_re, REG_EXTENDED) != 0) {
        die("cannot compile regex %s", whitelist_re);
        err = sc_error_init(SC_LIBSNAP_DOMAIN, SC_BUG, "cannot compile regular expression %s", whitelist_re);
        goto out;
    }

    // First capture is for verifying the full security tag, second capture
    // for verifying the snap_name is correct for this security tag.
    regmatch_t matches[2];
    int status = regexec(&re, security_tag, sizeof matches / sizeof *matches, matches, 0);
    regfree(&re);

    // Fail if no match or if snap name wasn't captured in the 2nd match group.
    if (status != 0 || matches[1].rm_so < 0) {
        err = sc_error_init_simple("invalid security tag %s", security_tag);
        goto out;
    }

out:
    return sc_error_forward(errorp, err);
}

int prog_args_parse(prog_args *args, int *argcp, char ***argvp, sc_error **errorp) {
    sc_error *err = NULL;

    /* Sanity check arguments. */
    if (args == NULL) {
        err = sc_error_init_api_misuse("args cannot be NULL");
        goto out;
    }
    /* Initialize args so that they are safe for cleanup. */
    args->security_tag = NULL;
    args->cgroup_name = NULL;
    args->is_version_query = false;

    /* Sanity check remaining arguments. */
    if (argcp == NULL || argvp == NULL) {
        err = sc_error_init_api_misuse("argcp and argvp cannot be NULL");
        goto out;
    }
    /* Use dereferenced versions of argcp and argvp for convenience. */
    int argc = *argcp;
    char **const argv = *argvp;
    if (argc == 0) {
        err = sc_error_init_api_misuse("argc cannot be zero");
        goto out;
    }
    if (argv == NULL) {
        err = sc_error_init_api_misuse("argv cannot be zero");
        goto out;
    }
    /* Sanity check each element of argv[]. */
    for (int i = 0; i < argc; ++i) {
        if (argv[i] == NULL) {
            err = sc_error_init_api_misuse("argv[%d] cannot be NULL", i);
            goto out;
        }
    }

    /* Parse option switches. */
    int optind;
    for (optind = 1; optind < argc; ++optind) {
        /* Look at all the options switches that start with the minus sign ('-') */
        if (argv[optind][0] != '-') {
            /* On first non-switch argument break the loop. The next loop looks
             * just for non-option arguments. This ensures that options and
             * positional arguments cannot be mixed. */
            break;
        }
        /* Handle option switches. */
        if (strcmp(argv[optind], "--version") == 0) {
            args->is_version_query = true;
            /* NOTE: --version short-circuits the parser to finish. */
            goto done;
        } else {
            /* Report unhandled option switches. */
            err = sc_error_init(PROG_ARGS_DOMAIN, PROG_ARGS_EUSAGE,
                                "Usage: snap-update-device-cgroup <cgroup-path> <security-tag>\n"
                                "\n"
                                "unrecognized command line option: %s",
                                argv[optind]);
            goto out;
        }
    }

    /* Parse positional arguments.
     * NOTE: optind is not reset, we just continue from where we left off in
     * the loop above. */
    for (; optind < argc; ++optind) {
        /* The first positional argument becomes the cgroup path. */
        if (args->cgroup_name == NULL) {
            args->cgroup_name = sc_strdup(argv[optind]);
            continue;
        }
        /* The second positional argument becomes the security tag. */
        if (args->security_tag == NULL) {
            args->security_tag = sc_strdup(argv[optind]);
            /* No more positional arguments are required. */
            break;
        }
    }

    /* Verify that all mandatory positional arguments are present. */
    if (args->cgroup_name == NULL) {
        err = sc_error_init(PROG_ARGS_DOMAIN, PROG_ARGS_EUSAGE,
                            "Usage: snap-update-device-cgroup <cgroup-name> <security-tag>\n"
                            "\n"
                            "cgroup path was not provided");
        goto out;
    }
    /* TODO: validate cgroup_name. */
    if (args->security_tag == NULL) {
        err = sc_error_init(PROG_ARGS_DOMAIN, PROG_ARGS_EUSAGE,
                            "Usage: snap-update-device-cgroup <cgroup-name> <security-tag>\n"
                            "\n"
                            "application or hook security tag was not provided");
        goto out;
    }
    if (check_snap_security_tag(args->security_tag, &err) < 0) {
        goto out;
    }

    int i;
done:
    /* "shift" the argument vector left, except for argv[0], to "consume" the
     * arguments that were scanned / parsed correctly. */
    for (i = 1; optind + i < argc; ++i) {
        argv[i] = argv[optind + i];
    }
    argv[i] = NULL;

    /* Write the updated argc back, argv is never modified. */
    *argcp = argc - optind;

out:
    /* Reset the state in case of an error. */
    if (err != NULL) {
        prog_args_cleanup(args);
    }
    return sc_error_forward(errorp, err);
}

void prog_args_free(prog_args *args) {
    if (args != NULL) {
        sc_cleanup_string(&args->cgroup_name);
        sc_cleanup_string(&args->security_tag);
    }
}

void prog_args_cleanup(prog_args *ptr) {
    if (ptr != NULL) {
        prog_args_free(ptr);
    }
}
