#include <jansson.h>

#include <stdio.h>
#include <sys/time.h>
#include <sys/types.h>
#include <strings.h>
#include <errno.h>
#include <stdlib.h>
#include <getopt.h>
#include <time.h>
#include <stdarg.h>
#include <assert.h>
#include <unistd.h>
#include <pwd.h>

#include <curl/curl.h>

// TODO:
// 	1. Use getpwnam() and getppid() instead of relying on the shell to do this

static int chist_timestamp(char *buffer, size_t bufsz) {
	struct timeval tv;
	if (gettimeofday(&tv, NULL) != 0) {
		perror("gettimeofday");
		assert(0);
		return -1;
	}

	struct tm *tm_info = localtime(&tv.tv_sec);
	if (!tm_info) {
		assert(tm_info);
		return -1;
	}

	int n1 = strftime(buffer, bufsz, "%Y-%m-%dT%H:%M:%S", tm_info);
	if (n1 <= 0) {
		assert(n1 > 0);
		return -1;
	}

	// Append microseconds
	int n2 = snprintf(&buffer[n1], bufsz-n1, ".%06d", tv.tv_usec);
	if (n2 <= 0) {
		assert(n2 > 0);
		return -1;
	}

	return n1 + n2;
}

void __attribute__((__noinline__)) chist_log_impl(
	const char *file,
	int line,
	const char *level,
	const char *format,
	...
) __attribute__((__format__(__printf__, 4, 5)));

// TODO: log to file
void chist_log_impl(const char *file, int line, const char *level, const char *format, ...) {
	char buf[32];
	if (chist_timestamp(buf, sizeof(buf)) <= 0) {
		strcpy(buf, "TIMESTAMP_ERROR");
	}

	if (level == NULL) {
		level = "INFO";
	}
	fprintf(stderr, "%s\t%s\t%s:%d\t", buf, level, file, line);
	va_list args;
	va_start(args, format);
	vfprintf(stderr, format, args);
	va_end(args);
}

#define chist_fatal(format, ...)                                             \
	do {                                                                     \
		chist_log_impl(__FILE__, __LINE__, "FATAL", format,  ##__VA_ARGS__); \
		exit(EXIT_FAILURE);                                                  \
	} while (0)

#define chist_log(format, ...)                                              \
	do {                                                                    \
		chist_log_impl(__FILE__, __LINE__, "INFO", format,  ##__VA_ARGS__); \
	} while (0)

char *current_user() {
	errno = 0;
	struct passwd *pw = getpwuid(getuid());
	assert(pw);
	return pw ? strndup(pw->pw_name, 512) : NULL;
}

void usage() {
	const char *name = "chist-client";
	fprintf(stderr,
		"Usage %s: [OPTION]... [HISTORY_ID] [COMMAND] [ARGS]...\n"
		"\nRequired options:\n"
		"  -s, --session\tterminal session id\n"
		"  -c, --status-code\tcommand status/exit code\n",
		name);
}

// TODO: remove if not used
struct memory_buffer {
	char   *data;
	size_t size;
};

int main(int argc, char *argv[]) {
	struct option longopts[] = {
		// {"user", required_argument, NULL, 'u'},
		{"session", required_argument, NULL, 's'},
		// {"ppid", required_argument, NULL, 'p'},
		{"status-code", required_argument, NULL, 'c'},
		{NULL, 0, NULL, 0},                // zero pad end
	};

	char *user = current_user();
	if (user == NULL) {
		chist_fatal("error: lookup user: %s", strerror(errno));
	}
	pid_t ppid = getppid();

	char *session = NULL;
	long status = 0;

	int ch;
	int opt_index = 0;
	while ((ch = getopt_long(argc, argv, "d", longopts, &opt_index)) != -1) {
		switch (ch) {
		// case 'u':
		// 	user = strndup(optarg, 512);
		// 	break;
		case 's':
			session = strndup(optarg, 512);
			break;
		// case 'p':
		// 	ppid = strtol(optarg, (char **)NULL, 10);
		// 	if (ppid <= 0) {
		// 		if (errno) {
		// 			chist_fatal("error: parsing 'ppid' argument: %s\n", strerror(errno));
		// 		}
		// 		chist_fatal("error: non-positive 'ppid' argument: %li\n", ppid);
		// 	}
		// 	break;
		case 'c':
			status = strtol(optarg, (char **)NULL, 10);
			if (status == 0 && errno != 0) {
				chist_fatal("error: parsing 'status' argument: %s\n", strerror(errno));
			}
			if (status < 0) {
				chist_fatal("error: non-positive 'status' argument: %li\n", status);
			}
			break;
		default:
			chist_fatal("error: invalid argument: %c\n", ch);
		}
	}
	argc -= optind;
	argv += optind;

	// if (user == NULL) {
	// 	chist_fatal("error: missing required argument: USER\n");
	// }
	if (session == NULL) {
		chist_fatal("error: missing required argument: SESSION\n");
	}
	if (argc < 1) {
		chist_fatal("error: missing COMMAND\n");
	}

	long long history_id = strtoll(argv[0], (char **)NULL, 10);
	if (history_id <= 0) {
		if (errno) {
			chist_fatal("error: parsing 'history_id' argument: %s\n", strerror(errno));
		}
		chist_fatal("error: non-positive 'history_id' argument: %lli\n", history_id);
	}

	argc -= 1;
	argv += 1;

	json_t *obj = json_object();
	if (!obj) {
		chist_fatal("fatal: failed to allocate JSON object\n");
	}
	if (json_object_set_new_nocheck(obj, "session_id", json_string(session)) != 0) {
		chist_fatal("error: setting: %s\n", "ppid");
	}
	if (json_object_set_new_nocheck(obj, "ppid", json_integer(ppid)) != 0) {
		chist_fatal("error: setting: %s\n", "ppid");
	}
	if (json_object_set_new_nocheck(obj, "status_code", json_integer(status)) != 0) {
		chist_fatal("error: setting: %s\n", "status_code");
	}
	if (json_object_set_new_nocheck(obj, "history_id", json_integer(history_id)) != 0) {
		chist_fatal("error: setting: %s\n", "history_id");
	}
	json_t *args = json_array();
	if (!args) {
		chist_fatal("fatal: failed to allocate JSON array\n");
	}
	for (int i = 0; i < argc; i++) {
		if (json_array_append_new(args, json_string(argv[i])) != 0) {
			chist_fatal("error: appending to array\n");
		}
	}
	if (json_object_set_new_nocheck(obj, "command", args) != 0) {
		chist_fatal("error: setting: %s\n", "command");
	}

	// TODO: destroy JSON object
	char *request_data = json_dumps(obj, JSON_ENSURE_ASCII|JSON_COMPACT);
	if (!request_data) {
		chist_fatal("json_dumps failed");
	}
	FILE *request_file = fmemopen(request_data, strlen(request_data), "r");
	if (!request_file) {
		chist_fatal("fmemopen: %s\n", strerror(errno));
	}

	// WARN
	// json_dumpf(obj, stdout, JSON_INDENT(4));

	CURL *curl_handle = curl_easy_init();
	if (!curl_handle) {
		chist_fatal("failed to initialize curl session\n");
	}

	// WARN
	const char * const server_socket = "/Users/cvieth/.local/share/histdb/socket/sock.sock";

	// WARN
	#define setopt(handle, opt, value)                                                     \
		do {                                                                               \
			CURLcode _res = curl_easy_setopt(handle, opt, value);                          \
			if (_res != CURLE_OK) {                                                        \
				chist_fatal("curl_easy_setopt: %s: %s\n", #opt, curl_easy_strerror(_res)); \
			}                                                                              \
		} while (0);

	setopt(curl_handle, CURLOPT_UNIX_SOCKET_PATH, server_socket);
	setopt(curl_handle, CURLOPT_URL, "http://unix/reflect");
	setopt(curl_handle, CURLOPT_READDATA, (void *)request_file);

	// FILE *response_file = open_memstream(char **__bufp, size_t *__sizep)

	printf("DONE!\n");
	return 0;
}

