/*
 * Copyright (C) 2024 Canonical Ltd
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
 * along with this program.  If not, see <http://www.org/licenses/>.
 *
 */

/**
 * snap-confine-workload
 *
 * A thin binary that applies security tags (AppArmor, seccomp) and executes
 * a workload binary. Unlike snap-confine, it does NOT set up:
 *   - mount namespaces
 *   - device cgroups
 *   - freezer cgroups
 *
 * This is because workloads run inside the snap's sandbox which is already
 * set up by snap-confine + snap-exec for the parent process. Device cgroup
 * enforcement is inherited from the parent's cgroup tree (eBPF-based on
 * cgroup v2).
 */

#ifdef HAVE_CONFIG_H
#include "config.h"
#endif

#include <errno.h>
#include <fcntl.h>
#include <sched.h>
#include <signal.h>
#include <stdbool.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <sys/prctl.h>
#include <sys/stat.h>
#include <sys/types.h>
#include <unistd.h>

#include "../libsnap-confine-private/apparmor-support.h"
#include "../libsnap-confine-private/cleanup-funcs.h"
#include "../libsnap-confine-private/error.h"
#include "../libsnap-confine-private/snap.h"
#include "../libsnap-confine-private/string-utils.h"
#include "../libsnap-confine-private/tool.h"
#include "../libsnap-confine-private/utils.h"
#include "seccomp-support.h"

#define SNAP_CONFINE_WORKLOAD_USAGE \
    "snap-confine-workload <security-tag> <executable> [args...]\n"

struct sc_workload_args {
    char *security_tag;
    char *executable;
};

static struct sc_workload_args *sc_workload_parse_args(int *argcp, char ***argvp, sc_error **errorp) {
    struct sc_workload_args *args = NULL;
    sc_error *err = NULL;
    int argc = *argcp;
    char **const argv = *argvp;

    if (argcp == NULL || argvp == NULL) {
        err = sc_error_init("args", 1, "cannot parse arguments, argcp or argvp is NULL");
        goto out;
    }
    if (argc < 3) {
        err = sc_error_init("args", 1, "Usage: %s\n\nsecurity-tag and executable are required",
                            SNAP_CONFINE_WORKLOAD_USAGE);
        goto out;
    }

    args = calloc(1, sizeof *args);
    if (args == NULL) {
        die("cannot allocate memory for command line arguments object");
    }

    args->security_tag = sc_strdup(argv[1]);
    args->executable = sc_strdup(argv[2]);

    /* Shift remaining arguments */
    int remaining = argc - 3;
    for (int i = 0; i < remaining; i++) {
        argv[i + 1] = argv[i + 3];
    }
    argv[remaining + 1] = NULL;

    *argcp = remaining + 1;

out:
    if (err != NULL) {
        free(args->security_tag);
        free(args->executable);
        free(args);
        args = NULL;
    }
    sc_error_forward(errorp, err);
    return args;
}

int main(int argc, char **argv) {
    sc_error *err = NULL;

    /* Parse command line arguments */
    struct sc_workload_args *args SC_CLEANUP(sc_cleanup_args_workload) = NULL;
    args = sc_workload_parse_args(&argc, &argv, &err);
    sc_die_on_error(err);

    /* Validate security tag */
    const char *security_tag = args->security_tag;
    const char *executable = args->executable;

    /* Extract snap instance name from security tag (format: snap.<instance>.<type>) */
    /* For workload tags, the format is: snap.<instance>.workload.<name> */
    /* We need to extract the snap instance name to validate against */
    const char *snap_instance_env = getenv("SNAP_INSTANCE_NAME");
    if (snap_instance_env == NULL) {
        die("SNAP_INSTANCE_NAME is not set");
    }

    if (!sc_security_tag_validate(security_tag, snap_instance_env, NULL)) {
        die("security tag %s not allowed for instance %s", security_tag, snap_instance_env);
    }

    /* Check if this is a workload security tag */
    if (!sc_is_workload_security_tag(security_tag)) {
        die("security tag %s is not a workload tag", security_tag);
    }

    /* Drop all inherited capabilities since we don't need them */
    if (prctl(PR_SET_NO_NEW_PRIVS, 1, 0, 0, 0) < 0) {
        die("cannot set PR_SET_NO_NEW_PRIVS");
    }

    /* Apply seccomp profile for the workload security tag */
    sc_apply_seccomp_profile_for_security_tag(security_tag);

    /* Set AppArmor profile as exec transition */
    struct sc_apparmor aa;
    sc_init_apparmor_support(&aa);
    if (aa.mode != SC_AA_NOT_APPLICABLE) {
        sc_maybe_aa_change_onexec(&aa, security_tag);
    }

    /* Reset PATH to a safe value inside the snap environment */
    setenv("PATH",
           "/usr/local/sbin:"
           "/usr/local/bin:"
           "/usr/sbin:"
           "/usr/bin:"
           "/sbin:"
           "/bin",
           1);

    /* Reset TMPDIRs */
    const char *tmpd[] = {"TMPDIR", "TEMPDIR", NULL};
    for (int i = 0; tmpd[i] != NULL; i++) {
        if (setenv(tmpd[i], "/tmp", 1) != 0) {
            die("cannot set environment variable '%s'", tmpd[i]);
        }
    }

    /* Execute the workload binary */
    debug("execv(%s, %s...)", executable, argv[1]);
    for (int i = 1; i < argc; i++) {
        debug(" argv[%i] = %s", i, argv[i]);
    }

    char *const *exec_argv = &argv[1];
    exec_argv[0] = (char *)executable;
    execv(executable, exec_argv);
    perror("execv failed");
    return 1;
}

void sc_cleanup_args_workload(struct sc_workload_args **ptr) {
    if (ptr && *ptr) {
        free((*ptr)->security_tag);
        (*ptr)->security_tag = NULL;
        free((*ptr)->executable);
        (*ptr)->executable = NULL;
        free(*ptr);
        *ptr = NULL;
    }
}

bool sc_is_workload_security_tag(const char *security_tag) {
    /* Check if the security tag ends with .workload.<name> */
    const char *suffix = ".workload.";
    size_t tag_len = strlen(security_tag);
    size_t suffix_len = strlen(suffix);

    if (tag_len <= suffix_len) {
        return false;
    }

    /* Check if tag ends with ".workload." */
    if (strcmp(security_tag + tag_len - suffix_len, suffix) != 0) {
        return false;
    }

    /* Check that there is a workload name after the suffix */
    const char *workload_name = security_tag + tag_len - suffix_len + suffix_len;
    if (strlen(workload_name) == 0) {
        return false;
    }

    /* Validate the workload name contains only valid characters */
    for (const char *p = workload_name; *p != '\0'; p++) {
        if (!((*p >= 'a' && *p <= 'z') || (*p >= 'A' && *p <= 'Z') ||
              (*p >= '0' && *p <= '9') || *p == '-' || *p == '_')) {
            return false;
        }
    }

    return true;
}
