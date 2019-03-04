// For AT_EMPTY_PATH and O_PATH
#define _GNU_SOURCE

#include "cgroup-support.h"

#include <errno.h>
#include <stdio.h>
#include <string.h>
#include <sys/stat.h>
#include <sys/types.h>
#include <unistd.h>

#include "cleanup-funcs.h"
#include "utils.h"

#define CGROUP_PROCS "cgroup.procs"

void sc_cgroup_create_and_join(const char *parent, const char *name, pid_t pid) {
    int parent_fd SC_CLEANUP(sc_cleanup_close) = -1;
    parent_fd = open(parent, O_PATH | O_DIRECTORY | O_NOFOLLOW | O_CLOEXEC);
    if (parent_fd < 0) {
        die("cannot open cgroup hierarchy %s", parent);
    }
    if (mkdirat(parent_fd, name, 0755) < 0 && errno != EEXIST) {
        die("cannot create cgroup hierarchy %s/%s", parent, name);
    }
    int hierarchy_fd SC_CLEANUP(sc_cleanup_close) = -1;
    hierarchy_fd = openat(parent_fd, name, O_PATH | O_DIRECTORY | O_NOFOLLOW | O_CLOEXEC);
    if (hierarchy_fd < 0) {
        die("cannot open cgroup hierarchy %s/%s", parent, name);
    }
    // Since we may be running from a setuid but not setgid executable, ensure
    // that the group and owner of the hierarchy directory is root.root.
    if (fchownat(hierarchy_fd, "", 0, 0, AT_EMPTY_PATH) < 0) {
        die("cannot change owner of cgroup hierarchy %s/%s to root.root", parent, name);
    }
    int procs_fd SC_CLEANUP(sc_cleanup_close) = -1;
    procs_fd = openat(hierarchy_fd, CGROUP_PROCS, O_WRONLY | O_NOFOLLOW | O_CLOEXEC);
    if (procs_fd < 0) {
        die("cannot open file %s/%s/" CGROUP_PROCS, parent, name);
    }
    FILE *stream SC_CLEANUP(sc_cleanup_file) = NULL;
    stream = fdopen(procs_fd, "w");
    if (stream == NULL) {
        die("cannot open stream of %s/%s/" CGROUP_PROCS, parent, name);
    }
    // stream now owns procs_fd
    procs_fd = -1;
    // Write the process (task) number to the cgroup.procs file. Linux task IDs
    // are limited to 2^29 so a long int is enough to represent it.  See
    // include/linux/threads.h in the kernel source tree for details.
    int n = fprintf(stream, "%ld", (long)pid);
    if (n < 0) {
        die("cannot move process %ld to cgroup hierarchy %s/%s", (long)pid, parent, name);
    }
    if (fflush(stream) == EOF) {
        die("cannot flush buffer");
    }
    debug("moved process %ld to cgroup hierarchy %s/%s", (long)pid, parent, name);
}
