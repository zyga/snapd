/*
 * Copyright (C) 2026 Canonical Ltd
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

/*
 * snap-container-workload is a small security-transition wrapper. It joins a
 * set of Linux namespaces, applies AppArmor/SELinux profiles, loads a
 * seccomp BPF filter, and then execs a target program.
 *
 * See cmd/snap-container-workload/DESIGN.md for the full design rationale.
 */
#ifdef HAVE_CONFIG_H
#include "config.h"
#endif

#include <ctype.h>
#include <errno.h>
#include <fcntl.h>
#include <limits.h>
#include <sched.h>
#include <signal.h>
#include <stdbool.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <sys/capability.h>
#include <sys/ioctl.h>
#include <sys/stat.h>
#include <sys/types.h>
#include <sys/wait.h>
#include <unistd.h>

#ifdef HAVE_SELINUX
#include "../libsnap-confine-private/selinux-support.h"
#endif

#include "../libsnap-confine-private/apparmor-support.h"
#include "../libsnap-confine-private/cleanup-funcs.h"
#include "../libsnap-confine-private/privs.h"
#include "../libsnap-confine-private/seccomp-support.h"
#include "../libsnap-confine-private/snap.h"
#include "../libsnap-confine-private/string-utils.h"
#include "../libsnap-confine-private/utils.h"

#ifndef NS_GET_NSTYPE
/* linux/nsfs.h, kernel 5.6+. Define locally in case the build headers are
 * older than that -- the ioctl is used only as a best-effort sanity check. */
#include <linux/ioctl.h>
#define NS_GET_NSTYPE _IO(0xb7, 0x1)
#endif

/* Namespace types supported by --<type>-ns=, ordered by the FIXED internal
 * join priority described in DESIGN.md ("Processing Order — Fixed Internal
 * Sequence"), with one exception: SC_NS_USER is intentionally listed first
 * (matching the design's priority table) but is *not* joined in that early
 * position by sc_join_namespaces() below. See the comment above
 * sc_join_namespaces() for the reconciliation of the two conflicting
 * ordering requirements called out in DESIGN.md. */
enum {
    SC_NS_USER = 0,
    SC_NS_PID,
    SC_NS_CGROUP,
    SC_NS_IPC,
    SC_NS_UTS,
    SC_NS_NET,
    SC_NS_MNT,
    SC_NS_TYPE_COUNT,
};

typedef struct {
    const char *name;
    int clone_flag;
} sc_ns_type_info;

static const sc_ns_type_info sc_ns_types[SC_NS_TYPE_COUNT] = {
    [SC_NS_USER] = {"user", CLONE_NEWUSER},       [SC_NS_PID] = {"pid", CLONE_NEWPID},
    [SC_NS_CGROUP] = {"cgroup", CLONE_NEWCGROUP}, [SC_NS_IPC] = {"ipc", CLONE_NEWIPC},
    [SC_NS_UTS] = {"uts", CLONE_NEWUTS},          [SC_NS_NET] = {"net", CLONE_NEWNET},
    [SC_NS_MNT] = {"mnt", CLONE_NEWNS},
};

/* Non-user types joined early, in this fixed order (see DESIGN.md table). */
static const int sc_early_join_order[] = {SC_NS_PID, SC_NS_CGROUP, SC_NS_IPC, SC_NS_UTS, SC_NS_NET, SC_NS_MNT};

#define SC_MAX_NS_ENTRIES 32

typedef struct {
    int type;          /* index into sc_ns_types */
    const char *value; /* fd number (as string) or filesystem path */
} sc_ns_entry;

static void sc_print_usage_and_die(const char *msg) {
    if (msg != NULL) {
        fprintf(stderr, "snap-container-workload: %s\n", msg);
    }
    fprintf(stderr, "usage: snap-container-workload --<type>-ns=... TAG [--] EXEC [ARGS...]\n");
    exit(2);
}

/* sc_parse_ns_option checks if "arg" has the form "--<type>-ns=<value>" and,
 * if so, fills in *type_name (pointer into arg, NOT NUL terminated at the
 * right spot -- length is *type_name_len) and *value (pointer into arg,
 * proper NUL terminated suffix). Returns false if arg doesn't match the
 * pattern at all (i.e. it's not a namespace option). */
static bool sc_parse_ns_option(const char *arg, const char **type_name, size_t *type_name_len, const char **value) {
    static const char prefix[] = "--";
    static const char suffix[] = "-ns=";

    size_t arg_len = strlen(arg);
    size_t prefix_len = sizeof(prefix) - 1;
    size_t suffix_len = sizeof(suffix) - 1;

    if (arg_len <= prefix_len + suffix_len) {
        return false;
    }
    if (strncmp(arg, prefix, prefix_len) != 0) {
        return false;
    }
    const char *marker = strstr(arg + prefix_len, suffix);
    if (marker == NULL) {
        return false;
    }
    const char *type_start = arg + prefix_len;
    size_t len = (size_t)(marker - type_start);
    if (len == 0) {
        return false;
    }
    *type_name = type_start;
    *type_name_len = len;
    *value = marker + suffix_len;
    return true;
}

static int sc_lookup_ns_type(const char *type_name, size_t type_name_len) {
    for (int i = 0; i < SC_NS_TYPE_COUNT; i++) {
        if (strlen(sc_ns_types[i].name) == type_name_len &&
            strncmp(sc_ns_types[i].name, type_name, type_name_len) == 0) {
            return i;
        }
    }
    return -1;
}

/* sc_derive_instance_and_component extracts the snap instance name and
 * optional component name out of a security tag, purely by lexical
 * inspection (it does not validate the tag; sc_security_tag_validate() does
 * that afterwards using the extracted values). The returned strings are
 * heap-allocated with strndup() and must be freed by the caller;
 * *component_out is left NULL when the tag has no component part. */
static void sc_derive_instance_and_component(const char *tag, char **instance_out, char **component_out) {
    static const char snap_prefix[] = "snap.";
    size_t prefix_len = sizeof(snap_prefix) - 1;

    *instance_out = NULL;
    *component_out = NULL;

    if (strncmp(tag, snap_prefix, prefix_len) != 0) {
        die("invalid security tag (must start with \"snap.\"): %s", tag);
    }
    const char *rest = tag + prefix_len;
    const char *dot = strchr(rest, '.');
    const char *plus = strchr(rest, '+');
    const char *end;
    if (plus != NULL && (dot == NULL || plus < dot)) {
        end = plus;
    } else {
        end = dot;
    }
    if (end == NULL || end == rest) {
        die("invalid security tag: %s", tag);
    }
    *instance_out = strndup(rest, (size_t)(end - rest));
    if (*instance_out == NULL) {
        die("cannot allocate memory");
    }
    if (*end == '+') {
        const char *comp_start = end + 1;
        const char *comp_end = strchr(comp_start, '.');
        if (comp_end == NULL || comp_end == comp_start) {
            die("invalid security tag (malformed component name): %s", tag);
        }
        *component_out = strndup(comp_start, (size_t)(comp_end - comp_start));
        if (*component_out == NULL) {
            die("cannot allocate memory");
        }
    }
}

static bool sc_str_is_all_digits(const char *s) {
    if (*s == '\0') {
        return false;
    }
    for (const char *p = s; *p != '\0'; p++) {
        if (!isdigit((unsigned char)*p)) {
            return false;
        }
    }
    return true;
}

/* sc_resolve_ns_fd resolves a --<type>-ns= value (fd number or path) to an
 * open file descriptor. *fd_needs_close indicates whether the caller is
 * responsible for closing the returned descriptor (paths are opened fresh
 * and must be closed; inherited numeric fds are left as-is). */
static int sc_resolve_ns_fd(const char *value, bool *fd_needs_close) {
    if (sc_str_is_all_digits(value)) {
        errno = 0;
        char *endptr = NULL;
        long fd_long = strtol(value, &endptr, 10);
        if (errno != 0 || endptr == value || *endptr != '\0' || fd_long < 0 || fd_long > INT_MAX) {
            die("invalid namespace file descriptor: %s", value);
        }
        int fd = (int)fd_long;
        if (fcntl(fd, F_GETFD) < 0) {
            die("cannot use namespace file descriptor %d", fd);
        }
        *fd_needs_close = false;
        return fd;
    }

    int fd = open(value, O_RDONLY | O_CLOEXEC);
    if (fd < 0) {
        die("cannot open namespace path %s", value);
    }
    *fd_needs_close = true;
    return fd;
}

static void sc_join_one_namespace(const sc_ns_entry *entry) {
    const sc_ns_type_info *info = &sc_ns_types[entry->type];

    bool fd_needs_close = false;
    int fd = sc_resolve_ns_fd(entry->value, &fd_needs_close);

    /* Best-effort namespace type validation (kernel 5.6+). Absence of the
     * ioctl (older kernels, or fd doesn't support it) is not fatal here --
     * setns(2) itself is the authoritative check. */
    int ns_type = 0;
    if (ioctl(fd, NS_GET_NSTYPE, &ns_type) == 0) {
        if (ns_type != info->clone_flag) {
            die("cannot join %s namespace: %s is not a %s namespace", info->name, entry->value, info->name);
        }
    }

    if (setns(fd, info->clone_flag) < 0) {
        int saved_errno = errno;
        if (fd_needs_close) {
            close(fd);
        }
        errno = saved_errno;
        die("cannot join %s namespace", info->name);
    }
    if (fd_needs_close) {
        close(fd);
    }
}

static void sc_join_namespaces_of_type(const sc_ns_entry *entries, size_t entry_count, int type) {
    for (size_t i = 0; i < entry_count; i++) {
        if (entries[i].type == type) {
            sc_join_one_namespace(&entries[i]);
        }
    }
}

/*
 * sc_join_namespaces implements the "Processing Order — Fixed Internal
 * Sequence" from DESIGN.md, with a deliberate reconciliation of the conflict
 * the design document itself calls out:
 *
 *   - The namespace-priority table says `user` must be joined *before* `pid`
 *     (so that UID/GID mappings apply to the eventual forked PID-namespace
 *     child).
 *   - The seccomp section's "Decision (chosen for now)" says the seccomp BPF
 *     filter must be loaded *before* the final setns(CLONE_NEWUSER) join,
 *     while CAP_SYS_ADMIN from the outer/privileged identity is still held.
 *
 * These two requirements only conflict if we insist on joining `user` in the
 * same pass as the other namespace types. They don't actually conflict in
 * practice: the *effect* that matters (UID mapping being visible in the
 * process that eventually fork()s to create the PID-namespace child, see
 * "Exec, Not Fork+Exec — Except When Joining a PID Namespace") only requires
 * `user` to be joined at any point *before* that fork() call, which itself
 * happens only after every other transition (namespaces, AppArmor/SELinux,
 * seccomp, capability drop) has completed.
 *
 * So: this function joins every *non*-user namespace type early, in the
 * fixed order (pid, cgroup, ipc, uts, net, mnt). The `user` namespace join is
 * deliberately deferred and performed by the caller after the seccomp filter
 * has been loaded (see sc_join_user_namespaces() below), immediately before
 * capabilities are dropped and well before any fork()/exec().
 */
static void sc_join_namespaces(const sc_ns_entry *entries, size_t entry_count) {
    for (size_t i = 0; i < SC_ARRAY_SIZE(sc_early_join_order); i++) {
        sc_join_namespaces_of_type(entries, entry_count, sc_early_join_order[i]);
    }
}

static void sc_join_user_namespaces(const sc_ns_entry *entries, size_t entry_count) {
    sc_join_namespaces_of_type(entries, entry_count, SC_NS_USER);
}

static bool sc_has_entries_of_type(const sc_ns_entry *entries, size_t entry_count, int type) {
    for (size_t i = 0; i < entry_count; i++) {
        if (entries[i].type == type) {
            return true;
        }
    }
    return false;
}

/* sc_ensure_cap_sys_admin_effective makes sure CAP_SYS_ADMIN is effective in
 * the current process, raising it from the permitted set if necessary. Dies
 * if CAP_SYS_ADMIN isn't even permitted -- loading a seccomp BPF program
 * requires it. */
static void sc_ensure_cap_sys_admin_effective(void) {
    cap_t caps SC_CLEANUP(sc_cleanup_cap_t) = cap_get_proc();
    if (caps == NULL) {
        die("cannot obtain current capabilities");
    }

    cap_flag_value_t effective = CAP_CLEAR;
    if (cap_get_flag(caps, CAP_SYS_ADMIN, CAP_EFFECTIVE, &effective) != 0) {
        die("cannot query CAP_SYS_ADMIN state");
    }
    if (effective == CAP_SET) {
        return;
    }

    cap_flag_value_t permitted = CAP_CLEAR;
    if (cap_get_flag(caps, CAP_SYS_ADMIN, CAP_PERMITTED, &permitted) != 0) {
        die("cannot query CAP_SYS_ADMIN state");
    }
    if (permitted != CAP_SET) {
        die("cannot load seccomp profile without CAP_SYS_ADMIN");
    }

    static const cap_value_t sys_admin[] = {CAP_SYS_ADMIN};
    if (cap_set_flag(caps, CAP_EFFECTIVE, SC_ARRAY_SIZE(sys_admin), sys_admin, CAP_SET) != 0) {
        die("cannot set capability flags");
    }
    if (cap_set_proc(caps) != 0) {
        die("cannot load seccomp profile without CAP_SYS_ADMIN");
    }
}

/* sc_drop_capabilities drops all capabilities from the process, except that
 * a real root process retains CAP_DAC_OVERRIDE (matching snap-confine's
 * behavior). This must be called after the seccomp filter has been loaded
 * (loading it requires CAP_SYS_ADMIN effective) and before exec. */
static void sc_drop_capabilities(void) {
    bool is_root = geteuid() == 0;

    cap_t caps SC_CLEANUP(sc_cleanup_cap_t) = cap_init();
    if (caps == NULL) {
        die("cannot allocate capabilities");
    }

    if (is_root) {
        static const cap_value_t dac_override[] = {CAP_DAC_OVERRIDE};
        if (cap_set_flag(caps, CAP_EFFECTIVE, SC_ARRAY_SIZE(dac_override), dac_override, CAP_SET) != 0 ||
            cap_set_flag(caps, CAP_PERMITTED, SC_ARRAY_SIZE(dac_override), dac_override, CAP_SET) != 0) {
            die("cannot set capability flags");
        }
    }

    if (cap_set_proc(caps) != 0) {
        die("cannot drop capabilities");
    }
}

/* Global state for the PID-namespace fork+wait+signal-forwarding path. Only
 * used when --pid-ns= was requested; see the comment above
 * sc_fork_exec_in_pid_ns(). */
static volatile sig_atomic_t sc_child_pid = 0;

static void sc_forward_signal(int signum) {
    pid_t child = sc_child_pid;
    if (child > 0) {
        kill(child, signum);
    }
}

static void sc_install_signal_forwarding(void) {
    struct sigaction sa;
    memset(&sa, 0, sizeof sa);
    sa.sa_handler = sc_forward_signal;
    sigemptyset(&sa.sa_mask);
    /* No SA_RESTART: we want wait loops etc. to be interrupted so the signal
     * is forwarded promptly. */
    static const int signals_to_forward[] = {SIGTERM, SIGINT, SIGHUP, SIGQUIT, SIGUSR1, SIGUSR2};
    for (size_t i = 0; i < SC_ARRAY_SIZE(signals_to_forward); i++) {
        if (sigaction(signals_to_forward[i], &sa, NULL) != 0) {
            die("cannot install signal handler");
        }
    }
}

/*
 * sc_fork_exec_in_pid_ns implements "Exec, Not Fork+Exec — Except When
 * Joining a PID Namespace" from DESIGN.md: setns(fd, CLONE_NEWPID) never
 * changes the *calling* process's own PID-namespace membership, only that of
 * children forked afterwards. So when --pid-ns= was requested we must fork,
 * exec the target in the child (which lands in the new PID namespace), and
 * have the parent forward signals and propagate the child's exit status.
 *
 * This function never returns: it exits with the appropriate status.
 */
static void sc_fork_exec_in_pid_ns(const char *exec_path, char **exec_argv, char **envp) {
    sc_install_signal_forwarding();

    pid_t child = fork();
    if (child < 0) {
        die("cannot fork");
    }
    if (child == 0) {
        /* Child: now (or soon to be) a member of the joined PID namespace. */
        execve(exec_path, exec_argv, envp);
        /* execve() only returns on error. */
        fprintf(stderr, "snap-container-workload: execv(%s): %s\n", exec_path, strerror(errno));
        _exit(127);
    }

    sc_child_pid = child;

    int status = 0;
    for (;;) {
        pid_t waited = waitpid(child, &status, 0);
        if (waited == child) {
            break;
        }
        if (waited < 0 && errno != EINTR) {
            die("cannot wait for child");
        }
        /* EINTR: a forwarded signal woke us up; loop and wait again. */
    }

    if (WIFEXITED(status)) {
        exit(WEXITSTATUS(status));
    }
    if (WIFSIGNALED(status)) {
        /* Re-raise the same fatal signal against ourselves so that our own
         * exit status/signal matches what the caller would have observed
         * from a direct exec(). */
        signal(WTERMSIG(status), SIG_DFL);
        raise(WTERMSIG(status));
        /* In case the signal was somehow ignored/blocked, fall back. */
        exit(128 + WTERMSIG(status));
    }
    /* Should not happen (stopped/continued statuses aren't produced by a
     * plain waitpid() without WUNTRACED/WCONTINUED), but handle gracefully. */
    exit(1);
}

int main(int argc, char **argv, char **envp) {
    if (argc < 2) {
        sc_print_usage_and_die(NULL);
    }

    sc_ns_entry ns_entries[SC_MAX_NS_ENTRIES];
    size_t ns_entry_count = 0;

    int i = 1;
    for (; i < argc; i++) {
        const char *arg = argv[i];
        if (strcmp(arg, "--") == 0) {
            i++;
            break;
        }
        const char *type_name = NULL;
        size_t type_name_len = 0;
        const char *value = NULL;
        if (!sc_parse_ns_option(arg, &type_name, &type_name_len, &value)) {
            /* Not a --<type>-ns= option: this is the start of TAG. */
            break;
        }
        int type = sc_lookup_ns_type(type_name, type_name_len);
        if (type < 0) {
            char buf[64];
            size_t len = type_name_len < sizeof(buf) - 1 ? type_name_len : sizeof(buf) - 1;
            memcpy(buf, type_name, len);
            buf[len] = '\0';
            die("unknown namespace type: %s", buf);
        }
        if (value[0] == '\0') {
            sc_print_usage_and_die("namespace option requires a value");
        }
        if (ns_entry_count >= SC_MAX_NS_ENTRIES) {
            die("too many namespace options (max %d)", SC_MAX_NS_ENTRIES);
        }
        ns_entries[ns_entry_count].type = type;
        ns_entries[ns_entry_count].value = value;
        ns_entry_count++;
    }

    if (i >= argc) {
        sc_print_usage_and_die("missing security tag");
    }
    const char *security_tag = argv[i];
    i++;

    if (i >= argc) {
        sc_print_usage_and_die("missing executable");
    }
    const char *exec_path = argv[i];
    char **exec_argv = &argv[i]; /* argv[argc] is guaranteed NULL by the C runtime */

    /* Validate security tag format. The instance/component values are
     * derived from the tag itself purely for the purpose of driving
     * sc_security_tag_validate()'s structural checks (it doesn't have an
     * external notion of what the "expected" snap is, unlike snap-confine
     * which learns the snap name from its own argv[0]/exec path). */
    char *derived_instance = NULL;
    char *derived_component = NULL;
    sc_derive_instance_and_component(security_tag, &derived_instance, &derived_component);
    if (!sc_security_tag_validate(security_tag, derived_instance, derived_component)) {
        die("invalid security tag: %s", security_tag);
    }
    free(derived_instance);
    free(derived_component);

    bool pid_ns_requested = sc_has_entries_of_type(ns_entries, ns_entry_count, SC_NS_PID);

    /* Step: initialize AppArmor support (queries our own current profile). */
    struct sc_apparmor apparmor;
    sc_init_apparmor_support(&apparmor);

    /* Step: join every non-user namespace in the fixed priority order. The
     * `user` namespace join is intentionally deferred; see the comment above
     * sc_join_namespaces(). */
    sc_join_namespaces(ns_entries, ns_entry_count);

    /* Step: apply AppArmor profile on next exec. The security tag IS the
     * AppArmor label. */
    sc_maybe_aa_change_onexec(&apparmor, security_tag);

#ifdef HAVE_SELINUX
    /* Step: apply SELinux exec context transition, if applicable. */
    sc_selinux_set_snap_execcon();
#endif

    /* Step: load the seccomp BPF filter for this security tag. This requires
     * CAP_SYS_ADMIN effective, and must happen before capabilities are
     * dropped and before the user namespace is joined (see the comment above
     * sc_join_namespaces() for why the latter is safe). */
    sc_ensure_cap_sys_admin_effective();
    sc_apply_seccomp_profile_for_security_tag(security_tag);

    /* Step: join the user namespace now that the seccomp filter is loaded
     * while we still held CAP_SYS_ADMIN from our original, privileged
     * identity. */
    sc_join_user_namespaces(ns_entries, ns_entry_count);

    /* Step: drop capabilities. Must happen after the seccomp load above. */
    sc_drop_capabilities();

    /* Step: exec (or fork+exec, if a PID namespace was joined). */
    if (pid_ns_requested) {
        sc_fork_exec_in_pid_ns(exec_path, exec_argv, envp);
        /* NOTREACHED */
    }

    execve(exec_path, exec_argv, envp);
    /* execve() only returns on error. */
    die("execv(%s)", exec_path);
    return 1;
}
