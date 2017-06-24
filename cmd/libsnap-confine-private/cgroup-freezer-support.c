// For AT_EMPTY_PATH and O_PATH
#define _GNU_SOURCE

#include "cgroup-freezer-support.h"

#include <errno.h>
#include <fcntl.h>
#include <limits.h>
#include <string.h>
#include <sys/stat.h>
#include <sys/types.h>
#include <unistd.h>

#include "../libsnap-confine-private/cleanup-funcs.h"
#include "../libsnap-confine-private/string-utils.h"
#include "../libsnap-confine-private/utils.h"

static const char *freezer_cgroup_dir = "/sys/fs/cgroup/freezer";

void sc_cgroup_freezer_join(const char *snap_name, pid_t pid)
{
	char buf[PATH_MAX];

	// Format the name of the cgroup hierarchy. 
	sc_must_snprintf(buf, sizeof buf, "snap.%s", snap_name);

	// Open the freezer cgroup directory.
	int cgroup_fd __attribute__ ((cleanup(sc_cleanup_close))) = -1;
	cgroup_fd = open(freezer_cgroup_dir,
			 O_PATH | O_DIRECTORY | O_NOFOLLOW | O_CLOEXEC);
	if (cgroup_fd < 0) {
		die("cannot open freezer cgroup (%s)", freezer_cgroup_dir);
	}
	// Create the freezer hierarchy for the given snap.
	if (mkdirat(cgroup_fd, buf, 0755) < 0 && errno != EEXIST) {
		die("cannot create freezer cgroup hierarchy for snap %s",
		    snap_name);
	}
	// Open the hierarchy directory for the given snap.
	int hierarchy_fd __attribute__ ((cleanup(sc_cleanup_close))) = -1;
	hierarchy_fd = openat(cgroup_fd, buf,
			      O_PATH | O_DIRECTORY | O_NOFOLLOW | O_CLOEXEC);
	if (hierarchy_fd < 0) {
		die("cannot open freezer cgroup hierarchy for snap %s",
		    snap_name);
	}
	// Since we are running from a setuid but not setgid executable, ensure
	// that the group and owner of the hierarchy directory is root.root. 
	if (fchownat(hierarchy_fd, "", 0, 0, AT_EMPTY_PATH) < 0) {
		die("cannot change owner of freezer cgroup hierarchy for snap %s to root.root", snap_name);
	}
	// Open the tasks file.
	int tasks_fd __attribute__ ((cleanup(sc_cleanup_close))) = -1;
	tasks_fd = openat(hierarchy_fd, "tasks",
			  O_WRONLY | O_NOFOLLOW | O_CLOEXEC);
	if (tasks_fd < 0) {
		die("cannot open tasks file for freezer cgroup hierarchy for snap %s", snap_name);
	}
	// Write the process (task) number to the tasks file.
	int n = sc_must_snprintf(buf, sizeof buf, "%ld", (long)pid);
	if (write(tasks_fd, buf, n) < 0) {
		die("cannot move process %ld to freezer cgroup hierarchy for snap %s", (long)pid, snap_name);
	}
	debug("moved process %ld to freezer cgroup hierarchy for snap %s",
	      (long)pid, snap_name);
}

void sc_cgroup_freezer_set_state(const char *snap_name, const char *state)
{
	char buf[PATH_MAX];

	// Format the name of the cgroup hierarchy. 
	sc_must_snprintf(buf, sizeof buf, "snap.%s", snap_name);

	// Open the freezer cgroup directory.
	int cgroup_fd __attribute__ ((cleanup(sc_cleanup_close))) = -1;
	cgroup_fd = open(freezer_cgroup_dir,
			 O_PATH | O_DIRECTORY | O_NOFOLLOW | O_CLOEXEC);
	if (cgroup_fd < 0) {
		die("cannot open freezer cgroup (%s)", freezer_cgroup_dir);
	}
	// Open the hierarchy directory for the given snap.
	int hierarchy_fd __attribute__ ((cleanup(sc_cleanup_close))) = -1;
	hierarchy_fd = openat(cgroup_fd, buf,
			      O_PATH | O_DIRECTORY | O_NOFOLLOW | O_CLOEXEC);
	if (hierarchy_fd < 0) {
		die("cannot open freezer cgroup hierarchy for snap %s",
		    snap_name);
	}
	// Open the freezer.state file.
	int state_fd __attribute__ ((cleanup(sc_cleanup_close))) = -1;
	state_fd = openat(hierarchy_fd, "freezer.state",
			  O_WRONLY | O_NOFOLLOW | O_CLOEXEC);
	if (state_fd < 0) {
		die("cannot open state file for freezer cgroup hierarchy for snap %s", snap_name);
	}
	// Write the desired state. 
	if (write(state_fd, state, strlen(state)) < 0) {
		die("cannot set state of cgroup hierarchy for snap %s to %s",
		    snap_name, state);
	}
	debug("set freezer cgroup hierarchy for snap %s to %s", snap_name,
	      state);
}

void sc_cgroup_freezer_frozen(const char *snap_name)
{
	sc_cgroup_freezer_set_state(snap_name, "FROZEN");
}

void sc_cgroup_freezer_thawed(const char *snap_name)
{
	sc_cgroup_freezer_set_state(snap_name, "THAWED");
}

void sc_cgroup_freezer_foreach_pid(const char *snap_name,
				   void (*f) (const char *pid,
					      struct sc_error ** errorp),
				   struct sc_error **errorp)
{
	char buf[PATH_MAX];

	// Format the name of the cgroup hierarchy. 
	sc_must_snprintf(buf, sizeof buf, "snap.%s", snap_name);

	// Open the freezer cgroup directory.
	int cgroup_fd __attribute__ ((cleanup(sc_cleanup_close))) = -1;
	cgroup_fd = open(freezer_cgroup_dir,
			 O_PATH | O_DIRECTORY | O_NOFOLLOW | O_CLOEXEC);
	if (cgroup_fd < 0) {
		die("cannot open freezer cgroup (%s)", freezer_cgroup_dir);
	}
	// Open the hierarchy directory for the given snap.
	int hierarchy_fd __attribute__ ((cleanup(sc_cleanup_close))) = -1;
	hierarchy_fd = openat(cgroup_fd, buf,
			      O_PATH | O_DIRECTORY | O_NOFOLLOW | O_CLOEXEC);
	if (hierarchy_fd < 0) {
		die("cannot open freezer cgroup hierarchy for snap %s",
		    snap_name);
	}
	// Open the cgroup.procs file. Note that it is not using a cleanup
	// attribute as it is closed via the FILE object below.
	int procs_fd = procs_fd = openat(hierarchy_fd, "cgroup.procs",
					 O_RDONLY | O_NOFOLLOW | O_CLOEXEC);
	if (procs_fd < 0) {
		die("cannot open cgroup.procs file for freezer cgroup hierarchy for snap %s", snap_name);
	}

	FILE *procs_file __attribute__ ((cleanup(sc_cleanup_file))) = NULL;
	procs_file = fdopen(procs_fd, "rt");
	if (procs_file == NULL) {
		die("cannot open cgroup.procs stream for freezer cgroup hierarchy for snap %s", snap_name);
	}
	// Read subsequent lines, each should contain one pid.
	char *line = NULL;
	size_t line_cap = 0;
	ssize_t line_len;
	struct sc_error *error = NULL;

	do {
		// Get each process ID and chomp the trailing newline.
		if ((line_len = getline(&line, &line_cap, procs_file)) < 0
		    && errno != 0) {
			die("cannot read process ID belonging to freezer cgroup hierachy for snap %s", snap_name);
		}
		if (line_len > 0 && line[line_len - 1] == '\n') {
			line[line_len - 1] = '\0';
			line_len -= 1;
		}
		if (line_len > 0) {
			f(line, &error);
		}
	} while (line_len > 0 || error != NULL);

	free(line);
	sc_error_forward(errorp, error);
}
