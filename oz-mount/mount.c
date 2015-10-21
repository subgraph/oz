#define _GNU_SOURCE
#include <stdlib.h>
#include <unistd.h>
#include <stdio.h>
#include <errno.h>
#include <string.h>

#include <fcntl.h>
#include <limits.h>
#include <sched.h>
#include <sys/types.h>
#include <sys/stat.h>

int checkProcessName(pid_t, char *);
int hasPrefix(const char *, const char *);

int enter_mount_namespace(void) {
	if (geteuid() != 0) {
		fprintf(stderr, "E Must run as root\n");
		return -1;
	}

	// Do some minimal verification to check that oz-daemon is the parent
	pid_t ppid = getppid();
	if (checkProcessName(ppid, "oz-daemon") != 0) {
		fprintf(stderr, "E unable to verify that oz-daemon is parent\n");
		return -1;
	}

	// Parse namespace pid from environment
	char *envv, *envvend;
	long nspid;
	envv = getenv("_OZ_NSPID");
	if (envv == NULL) {
		fprintf(stderr, "E unable to get namespace pid from environment\n");
		return -1;
	}
	errno = 0;
	nspid = strtol(envv, &envvend, 10);
	if ((errno == ERANGE && (nspid == LONG_MAX || nspid == LONG_MIN))
	|| (errno != 0 && nspid == 0)) {
		fprintf(stderr, "E unable to parse namespace pid from environment\n");
		return -1;
	}
	if (envvend == envv || nspid < 0) {
		fprintf(stderr, "E unable to parse namespace pid from environment\n");
		return -1;
	}

	// Verify that the target is an instance of oz-init
	if (checkProcessName(nspid, "oz-init") != 0) {
		fprintf(stderr, "E unable to verify that oz-init is the target\n");
		return -1;
	}

	char nspath[PATH_MAX];
	if (snprintf(nspath, PATH_MAX-1, "/proc/%ld/ns", nspid) < 0) {
		fprintf(stderr, "E unable to parse namespace path `/proc/%ld/ns`\n", nspid);
		return -1;
	}
	printf("D Opening: %s\n", nspath);

	// Start opening the namespace
	struct stat st;
	int tfd, fd;
	tfd = open(nspath, O_DIRECTORY | O_RDONLY);
	if (tfd == -1) {
		fprintf(stderr, "E failed to open child namespace\n");
		return -1;
	}
	// Symlinks on all namespaces exist for dead processes, but they can't be opened
	if (fstatat(tfd, "mnt", &st, AT_SYMLINK_NOFOLLOW) == -1) {
		if (errno == ENOENT) {
			fprintf(stderr, "E failed to open child namespace\n");
			return -1;
		}
	}
	fd = openat(tfd, "mnt", O_RDONLY);
	if (fd == -1) {
		fprintf(stderr, "E failed to open child mount namespace: %s\n", nspath);
		return -1;
	}
	// Set the namespace.
	if (setns(fd, 0) == -1) {
		fprintf(stderr, "E failed to setns for: %s\n", nspath);
		return -1;
	}
	close(fd);
}

int checkProcessName(pid_t pid, char *pname) {
	FILE *fp;
	char *line = NULL;
	char pproc[PATH_MAX];
	char cline[PATH_MAX];
	size_t len = 0;
	ssize_t read;

	if (snprintf(cline, PATH_MAX-1, "Name:	%s\n", pname) < 0) {
		return -1;
	}

	if (snprintf(pproc, PATH_MAX-1, "/proc/%ld/status", pid) < 0) {
		return -1;
	}

	fp = fopen(pproc, "r");
	if (fp == NULL) {
		return -1;
	}

	int retval = -1;
	while (retval == -1 && (read = getline(&line, &len, fp)) != -1) {
		if (hasPrefix(line, "Name:") >= 0) {
			retval = strcmp(cline, line);
			break;
		}
	}

	fclose(fp);
	if (line) {
		free(line);
	}

	return retval;
}

int hasPrefix(const char *str, const char *pre) {
	size_t lenpre = strlen(pre),
		lenstr = strlen(str);
	return lenstr < lenpre ? -1 : strncmp(pre, str, lenpre);
}
