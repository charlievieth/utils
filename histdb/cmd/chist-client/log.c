#include "log.h"

#include <stdio.h>
#include <string.h>
#include <time.h>
#include <stdarg.h>
#include <assert.h>

#include <sys/time.h>  // gettimeofday, timeval

int chist_log_level_parse(const char *str, enum log_level_t *lvl) {
	char upr[6]; // strlen("FATAL") + 1
	size_t n = strlen(str);
	if (!(strlen("INFO") <= n && n <= strlen("FATAL"))) {
		goto error;
	}
	for (int i = 0; i < (int)n; i++) {
		char c = str[i];
		if ('a' <= c && c <= 'z') {
			c -= 'a' - 'A';
		}
		upr[i] = c;
	}
	upr[n] = '\0';

	if (strcmp(upr, "DEBUG") == 0) {
		*lvl = LOG_LEVEL_DEBUG;
	} else if (strcmp(upr, "INFO") == 0) {
		*lvl = LOG_LEVEL_INFO;
	} else if (strcmp(upr, "WARN") == 0) {
		*lvl = LOG_LEVEL_WARN;
	} else if (strcmp(upr, "ERROR") == 0) {
		*lvl = LOG_LEVEL_ERROR;
	} else if (strcmp(upr, "FATAL") == 0) {
		*lvl = LOG_LEVEL_FATAL;
	} else {
		goto error;
	}

	return 0;

error:
	*lvl = LOG_LEVEL_INFO;
	return 1;
}

const char *chist_log_level_string(enum log_level_t lvl) {
	static const char *strs[] = {
		"DEBUG",
		"INFO",
		"WARN",
		"ERROR",
		"FATAL",
	};
	if (lvl <= sizeof(strs) / sizeof(strs[0])) {
		return strs[lvl];
	}
	return "UNKNOWN";
}

static int chist_timestamp(char *buffer, size_t bufsz)
	__attribute__((__nonnull__(1)));

static int chist_timestamp(char *buffer, size_t bufsz) {
	assert(bufsz >= 32);

	struct timeval tv;
	if (gettimeofday(&tv, NULL) != 0) {
		perror("gettimeofday");
		assert(0);
		return -1;
	}

	struct tm *tm_info = localtime(&tv.tv_sec);
	if (!tm_info) {
		perror("localtime");
		assert(tm_info);
		return -1;
	}

	int n;
	char format[27];

	n = snprintf(format, sizeof(format), "%%Y-%%m-%%dT%%H:%%M:%%S" ".%06d" "%%z",
		tv.tv_usec);
	assert(n == sizeof(format) - 1);

	n = strftime(buffer, bufsz, format, tm_info);
	assert(n > 0);
	return n;
}

static FILE **chist_log_output = NULL;

void chist_atexit() {
	FILE **fds = chist_log_output;
	if (fds) {
		FILE *fd;
		while ((fd = *fds) != NULL) {
			if (fd != stdout && fd != stderr) {
				fclose(fd);
			}
			*fds++ = NULL;
		}
	}
}

// int chist_add_log_output(FILE *fp) {
// 	size_t n = 0;
// 	if (!chist_log_output) {
// 		chist_log_output = calloc(2, sizeof(chist_log_output[0]));
// 	} else {
// 		while (chist_log_output[n++]) {
// 		};
// 		chist_log_output = realloc(chist_log_output, sizeof(chist_log_output[0]) * (n + 1));
// 	}
// 	chist_log_output[n] = fp;
// 	chist_log_output[n + 1] = NULL;
// 	return 0;
// }

static int chist_log_output_len() {
	int n = 0;
	if (chist_log_output) {
		while (chist_log_output[n]) {
			n++;
		};
	}
	return n;
}

static int append_log_output(FILE *fd) {
	assert(fd);

	size_t n = 0;
	if (!chist_log_output) {
		chist_log_output = calloc(3, sizeof(chist_log_output[0]));
		if (!chist_log_output) {
			return 1;
		}
	} else {
		FILE *f;
		while ((f = chist_log_output[n]) != NULL) {
			if (f == fd) {
				return 0;
			}
			n++;
		};
		FILE **fds = realloc(chist_log_output, sizeof(chist_log_output[0]) * (n + 2));
		if (!fds) {
			return 1;
		}
		chist_log_output = fds;
	}
	chist_log_output[n] = fd;
	chist_log_output[n + 1] = NULL;
	return 0;
}

int chist_add_log_output(const char *logname) {
	FILE *fd = fopen(logname, "w+");
	if (!fd) {
		return 1;
	}
	return append_log_output(fd);
}

/*
int init_log_files(const char *log_dir) {
	// chist_log_output[0] = stderr;

	char *name = NULL;
	asprintf(&name, "%s/client.log", log_dir);
	chist_log_output = fopen(name, "w+");
	free(name);

	if (!chist_log_output) {
		return 1; // WARN
	}

	return 0;
}
*/

// TODO: log to file
void chist_log_impl(const char *file, int line, enum log_level_t lvl, const char *format, ...) {
	char *bufp = NULL;
	size_t sizep;
	FILE *mstream = open_memstream(&bufp, &sizep);
	assert(mstream);
	if (!mstream) {
		return;
	}

	char ts[32];
	if (chist_timestamp(ts, sizeof(ts)) <= 0) {
		strcpy(ts, "TIMESTAMP_ERROR");
	}
	const char *level = chist_log_level_string(lvl);
	fprintf(mstream, "%s\t%s\t%s:%d\t", ts, level, file, line);

	va_list args;
	va_start(args, format);
	vfprintf(mstream, format, args);
	va_end(args);

	assert(fflush(mstream) == 0);
	if (sizep > 0 && bufp[sizep - 1] != '\n') {
		fputc('\n', mstream);
	}
	assert(fclose(mstream) == 0);

	fwrite(bufp, 1, sizep, stderr);

	if (bufp) {
		free(bufp);
	}
}

#include <stddef.h>
int main(int argc, char const *argv[]) {
	printf("1: len: %i\n", chist_log_output_len());
	append_log_output(stderr);
	printf("2: len: %i\n", chist_log_output_len());
	append_log_output(stdout);
	printf("3: len: %i\n", chist_log_output_len());
	append_log_output(stdout);
	printf("4: len: %i\n", chist_log_output_len());
	append_log_output(stdin);
	printf("4: len: %i\n", chist_log_output_len());
	return 0;


	FILE *fds[4] = { 0 };
	fds[0] = stdout;
	fds[1] = stderr;
	// fds[2] = stderr;

    const union { FILE *f; char c[sizeof(FILE*)]; } zero = { 0 };
    printf("union: %zu -- %s\n", sizeof(zero), zero.f == NULL ? "TRUE" : "FALSE");
	FILE **p = memmem(fds, sizeof(fds), &zero, sizeof(zero));

	ptrdiff_t n = (p - fds) + 1;
	printf("n: %zu\n", n);

	// printf("fds: %p\n", fds);
	// printf("p:   %p\n", p);
	// printf("x: %zu\n", (size_t)(fds) - (size_t)(p));

	// size_t n = 0;
	// while (fds[n]) {
	// 	n++;
	// };
	return 0;
}
