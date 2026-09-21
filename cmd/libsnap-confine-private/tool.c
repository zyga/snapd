/*
 * Copyright (C) 2018 Canonical Ltd
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

#ifdef HAVE_CONFIG_H
#include "config.h"
#endif

#include "tool.h"

#include <fcntl.h>
#include <libgen.h>
#include <string.h>
#include <sys/capability.h>
#include <sys/stat.h>
#include <sys/types.h>
#include <sys/wait.h>
#include <unistd.h>

#include "../libsnap-confine-private/apparmor-support.h"
#include "../libsnap-confine-private/cleanup-funcs.h"
#include "../libsnap-confine-private/string-utils.h"
#include "../libsnap-confine-private/utils.h"
#include "privs.h"

/**
 * sc_open_snapd_tool returns a file descriptor of the given internal executable.
 *
 * The executable is located based on the location of the currently executing process.
 * The returning file descriptor can be used with fexecve function, like in sc_call_snapd_tool.
 **/
static int sc_open_snapd_tool(const char *__null_terminated tool_name);

/* cap_free(3) takes the capability object by value, but SC_CLEANUP() hands the
 * cleanup function a pointer to the variable being cleaned up.  Adapt between
 * the two so that a cap_t can be used with SC_CLEANUP(). */
static void sc_cleanup_cap(cap_t *__unsafe_indexable ptr) {
    if (ptr != NULL && *ptr != NULL) {
        cap_free(*ptr);
    }
}

/* As above, but for the string returned by cap_to_name(3), which is also
 * released with cap_free(3). */
static void sc_cleanup_cap_name(const char *__single *__unsafe_indexable ptr) {
    if (ptr != NULL && *ptr != NULL) {
        cap_free((void *)*ptr);
    }
}

typedef struct {
    /* number of entries in @caps */
    size_t n_caps;
    const cap_value_t *__counted_by(n_caps) caps;
} sc_tool_privs;

/**
 * sc_call_snapd_tool calls a snapd tool by file descriptor.
 *
 * The idea with calling with an open file descriptor is to allow calling executables
 * across mount namespaces, where the executable may not be visible in the new filesystem
 * anymore. The caller establishes an open file descriptor in one namespace and later on
 * performs the call in another mount namespace.
 *
 * The environment vector has special support for expanding the string "SNAPD_DEBUG=x".
 * If such string is present, the "x" is replaced with either "0" or "1" depending on
 * the result of is_sc_debug_enabled().
 *
 * Identity indicates the set of effective uid/gid which are to be applied in
 * the child process right after forking.
 **/
static void sc_call_snapd_tool(int tool_fd, const char *__null_terminated tool_name, sc_tool_privs privs,
                               char *__null_terminated *__null_terminated argv,
                               char *__null_terminated *__null_terminated envp);

/**
 * sc_call_snapd_tool_with_apparmor calls a snapd tool by file descriptor,
 * possibly confining the program with a specific apparmor profile.
 **/
static void sc_call_snapd_tool_with_apparmor(int tool_fd, const char *__null_terminated tool_name,
                                             struct sc_apparmor *apparmor, const char *__null_terminated aa_profile,
                                             sc_tool_privs privs, char *__null_terminated *__null_terminated argv,
                                             char *__null_terminated *__null_terminated envp);

int sc_open_snap_update_ns(void) { return sc_open_snapd_tool("snap-update-ns"); }

void sc_call_snap_update_ns(int snap_update_ns_fd, const char *__null_terminated snap_name,
                            struct sc_apparmor *apparmor) {
    char *__unsafe_indexable snap_name_copy SC_CLEANUP(sc_cleanup_string) = NULL;
    snap_name_copy = sc_strdup(snap_name);

    char aa_profile[PATH_MAX] = {0};
    sc_must_snprintf(aa_profile, sizeof aa_profile, "snap-update-ns.%s", snap_name);

    char *__null_terminated argv[] __null_terminated = {
        "snap-update-ns",
        /* This tells snap-update-ns we are calling from snap-confine and locking is in place */
        "--from-snap-confine", __unsafe_forge_null_terminated(char *, snap_name_copy), NULL};
    char *__null_terminated envp[] __null_terminated = {"SNAPD_DEBUG=x", NULL};

    /* keep the current identity, privileges are carried over through capabilities */
    const cap_value_t caps[] = {
        CAP_DAC_OVERRIDE, /* poking around as a regular user */
        CAP_SYS_ADMIN,    /* mounts */
        CAP_CHOWN,        /* file ownership */
    };
    sc_tool_privs privs = {
        .n_caps = SC_ARRAY_SIZE(caps),
        .caps = caps,
    };
    sc_call_snapd_tool_with_apparmor(snap_update_ns_fd, "snap-update-ns", apparmor,
                                     __unsafe_forge_null_terminated(const char *, &aa_profile[0]), privs, argv, envp);
}

void sc_call_snap_update_ns_as_user(int snap_update_ns_fd, const char *__null_terminated snap_name,
                                    struct sc_apparmor *apparmor) {
    char *__unsafe_indexable snap_name_copy SC_CLEANUP(sc_cleanup_string) = NULL;
    snap_name_copy = sc_strdup(snap_name);

    char aa_profile[PATH_MAX] = {0};
    sc_must_snprintf(aa_profile, sizeof aa_profile, "snap-update-ns.%s", snap_name);

    const char *__null_terminated xdg_runtime_dir =
        __unsafe_forge_null_terminated(const char *, getenv("XDG_RUNTIME_DIR"));
    char xdg_runtime_dir_env[PATH_MAX + sizeof("XDG_RUNTIME_DIR=")] = {0};
    if (xdg_runtime_dir != NULL) {
        sc_must_snprintf(xdg_runtime_dir_env, sizeof(xdg_runtime_dir_env), "XDG_RUNTIME_DIR=%s", xdg_runtime_dir);
    }

    const char *__null_terminated snap_real_home =
        __unsafe_forge_null_terminated(const char *, getenv("SNAP_REAL_HOME"));
    char snap_real_home_env[PATH_MAX + sizeof("SNAP_REAL_HOME=")] = {0};
    if (snap_real_home != NULL) {
        sc_must_snprintf(snap_real_home_env, sizeof(snap_real_home_env), "SNAP_REAL_HOME=%s", snap_real_home);
    }

    char *__null_terminated argv[] __null_terminated = {
        "snap-update-ns",
        /* This tells snap-update-ns we are calling from snap-confine and locking is in place */
        "--from-snap-confine",
        /* This tells snap-update-ns that we want to process the per-user profile */
        "--user-mounts", __unsafe_forge_null_terminated(char *, snap_name_copy), NULL};
    char *__null_terminated envp[] __null_terminated = {
        /* SNAPD_DEBUG=x is replaced by sc_call_snapd_tool_with_apparmor
         * with either SNAPD_DEBUG=0 or SNAPD_DEBUG=1, see that function
         * for details. */
        "SNAPD_DEBUG=x", __unsafe_forge_null_terminated(char *, &xdg_runtime_dir_env[0]),
        __unsafe_forge_null_terminated(char *, &snap_real_home_env[0]), NULL};
    /* keep the current identity */
    const cap_value_t caps[] = {
        CAP_DAC_OVERRIDE, /* poking around as a regular user */
        CAP_SYS_ADMIN,    /* mounts */
        CAP_CHOWN,        /* file ownership */
    };
    sc_tool_privs privs = {
        .n_caps = SC_ARRAY_SIZE(caps),
        .caps = caps,
    };
    sc_call_snapd_tool_with_apparmor(snap_update_ns_fd, "snap-update-ns", apparmor,
                                     __unsafe_forge_null_terminated(const char *, &aa_profile[0]), privs, argv, envp);
}

int sc_open_snap_discard_ns(void) { return sc_open_snapd_tool("snap-discard-ns"); }

void sc_call_snap_discard_ns(int snap_discard_ns_fd, const char *__null_terminated snap_name) {
    char *__unsafe_indexable snap_name_copy SC_CLEANUP(sc_cleanup_string) = NULL;
    snap_name_copy = sc_strdup(snap_name);
    char *__null_terminated argv[] __null_terminated = {"snap-discard-ns", "--from-snap-confine",
                                                        __unsafe_forge_null_terminated(char *, snap_name_copy), NULL};
    /* SNAPD_DEBUG=x is replaced by sc_call_snapd_tool_with_apparmor with
     * either SNAPD_DEBUG=0 or SNAPD_DEBUG=1, see that function for details. */
    char *__null_terminated envp[] __null_terminated = {"SNAPD_DEBUG=x", NULL};
    /* keep the current identity, privileges are carried over through capabilities */
    const cap_value_t caps[] = {
        CAP_DAC_OVERRIDE, /* poking around as a regular user */
        CAP_SYS_ADMIN,    /* umount() */
        CAP_CHOWN,        /* file ownership */
    };
    sc_tool_privs privs = {
        .n_caps = SC_ARRAY_SIZE(caps),
        .caps = caps,
    };
    sc_call_snapd_tool(snap_discard_ns_fd, "snap-discard-ns", privs, argv, envp);
}

static int sc_open_snapd_tool(const char *__null_terminated tool_name) {
    // +1 is for the case where the link is exactly PATH_MAX long but we also
    // want to store the terminating '\0'. The readlink system call doesn't add
    // terminating null, but our initialization of buf handles this for us.
    char buf[PATH_MAX + 1] = {0};
    if (readlink("/proc/self/exe", buf, sizeof(buf) - 1) < 0) {
        die("cannot readlink /proc/self/exe");
    }
    if (buf[0] != '/') {  // this shouldn't happen, but make sure have absolute path
        die("readlink /proc/self/exe returned relative path");
    }
    // as we are looking up other tools relative to our own path, check
    // we are located where we think we should be - otherwise we
    // may have been hardlink'd elsewhere and then may execute the
    // wrong tool as a result
    if (!sc_is_expected_path(__unsafe_forge_null_terminated(const char *, &buf[0]))) {
        die("running from unexpected location: %s", buf);
    }
    char *__null_terminated dir_name = __unsafe_forge_null_terminated(char *, dirname(buf));
    int dir_fd SC_CLEANUP(sc_cleanup_close) = -1;
    dir_fd = open(dir_name, O_PATH | O_DIRECTORY | O_NOFOLLOW | O_CLOEXEC);
    if (dir_fd < 0) {
        die("cannot open path %s", dir_name);
    }
    int tool_fd = -1;
    tool_fd = openat(dir_fd, tool_name, O_PATH | O_NOFOLLOW | O_CLOEXEC);
    if (tool_fd < 0) {
        die("cannot open path %s/%s", dir_name, tool_name);
    }
    debug("opened %s executable as file descriptor %d", tool_name, tool_fd);
    return tool_fd;
}

static void sc_call_snapd_tool(int tool_fd, const char *__null_terminated tool_name, sc_tool_privs privs,
                               char *__null_terminated *__null_terminated argv,
                               char *__null_terminated *__null_terminated envp) {
    sc_call_snapd_tool_with_apparmor(tool_fd, tool_name, NULL, NULL, privs, argv, envp);
}

static void sc_call_snapd_tool_with_apparmor(int tool_fd, const char *__null_terminated tool_name,
                                             struct sc_apparmor *apparmor, const char *__null_terminated aa_profile,
                                             sc_tool_privs privs, char *__null_terminated *__null_terminated argv,
                                             char *__null_terminated *__null_terminated envp) {
    debug("calling snapd tool %s", tool_name);
    pid_t child = fork();
    if (child < 0) {
        die("cannot fork to run snapd tool %s", tool_name);
    }
    if (child == 0) {
        if (sc_cap_reset_ambient() != 0) {
            die("cannot reset ambient capabilities");
        }

        cap_t __single working SC_CLEANUP(sc_cleanup_cap) = __unsafe_forge_single(cap_t, cap_init());
        if (working == NULL) {
            die("cannot allocate capability set");
        }
        cap_clear_flag(working, CAP_EFFECTIVE);
        cap_set_flag(working, CAP_PERMITTED, privs.n_caps, privs.caps, CAP_SET);
        cap_set_flag(working, CAP_INHERITABLE, privs.n_caps, privs.caps, CAP_SET);

        if (cap_set_proc(working) != 0) {
            die("cannot set capabilities");
        }

        /* now that permitted and inheritable caps are set, we can also set ambient caps */
        for (size_t i = 0; i < privs.n_caps; i++) {
            if (sc_cap_set_ambient(privs.caps[i], CAP_SET) != 0) {
                const char *__single txt_cap SC_CLEANUP(sc_cleanup_cap_name) =
                    __unsafe_forge_single(const char *, cap_to_name(privs.caps[i]));
                die("cannot set ambient capability: %s", txt_cap);
            }
        }

        /* If the caller provided template environment entry for SNAPD_DEBUG
         * then expand it to the actual value. */
        for (char *__null_terminated *__null_terminated env = envp;
             /* Mama mia, that's a spicy meatball. */
             env != NULL && *env != NULL && **env != '\0'; env++) {
            if (sc_streq(*env, "SNAPD_DEBUG=x")) {
                /* NOTE: this is not released, on purpose. */
                char *__null_terminated entry = sc_strdup(*env);
                __null_terminated_to_indexable(entry)[strlen("SNAPD_DEBUG=x") - 1] = sc_is_debug_enabled() ? '1' : '0';
                *env = entry;
            }
        }
        /* Switch apparmor profile for the process after exec. */
        if (apparmor != NULL && aa_profile != NULL) {
            sc_maybe_aa_change_onexec(apparmor, aa_profile);
        }
        fexecve(tool_fd, argv, envp);
        die("cannot execute snapd tool %s", tool_name);
    } else {
        int status = 0;
        debug("waiting for snapd tool %s to terminate", tool_name);
        if (waitpid(child, &status, 0) < 0) {
            die("cannot get snapd tool %s termination status via waitpid", tool_name);
        }
        if (WIFEXITED(status) && WEXITSTATUS(status) != 0) {
            die("%s failed with code %i", tool_name, WEXITSTATUS(status));
        } else if (WIFSIGNALED(status)) {
            die("%s killed by signal %i", tool_name, WTERMSIG(status));
        }
        debug("%s finished successfully", tool_name);
    }
}
