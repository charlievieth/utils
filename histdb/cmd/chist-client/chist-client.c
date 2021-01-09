#include <jansson.h>

#include <stdio.h>
#include <sys/time.h>
#include <sys/types.h>
#include <sys/socket.h>
#include <sys/un.h>
#include <strings.h>
#include <ctype.h>
#include <stdbool.h>
#include <errno.h>
#include <stdlib.h>
#include <getopt.h>
#include <time.h>
#include <stdarg.h>
#include <assert.h>
#include <unistd.h>
#include <pwd.h>

#include <sys/param.h> // MAXPATHLEN

// TODO:
// 	1. Remove unused include stmts

static struct option longopts[] = {
	{"debug", no_argument, NULL, 'd'},
	{"help", no_argument, NULL, 'h'},
	{"session", required_argument, NULL, 0},
	{"status-code", required_argument, NULL, 0},
	{NULL, 0, NULL, 0},                // zero pad end
};

enum log_level_t {
	LOG_LEVEL_DEBUG,
	LOG_LEVEL_INFO,
	LOG_LEVEL_WARN,
	LOG_LEVEL_ERROR,
	LOG_LEVEL_FATAL
};

static enum log_level_t log_level = LOG_LEVEL_ERROR;

const char * chist_log_lvl_str(enum log_level_t lvl) {
	switch (lvl) {
	case LOG_LEVEL_DEBUG:
		return "DEBUG";
	case LOG_LEVEL_INFO:
		return "INFO";
	case LOG_LEVEL_WARN:
		return "WARN";
	case LOG_LEVEL_ERROR:
		return "ERROR";
	case LOG_LEVEL_FATAL:
		return "FATAL";
	default:
		return "UNKNOWN";
	}
}

static void usage(FILE *stream) {
	if (!stream) {
		stream = stderr;
	}
	const char *name = "chist-client";
	fprintf(stream,
		"Usage %s: [OPTION]... [HISTORY_ID] [COMMAND] [ARGS]...\n"
		"\nRequired options:\n"
		"  -d, --debug\tprint debug information\n"
		"  -c, --status-code\tcommand status/exit code\n"
		"  -s, --session\tterminal session id\n",
		name);
}

static int chist_timestamp(char *buffer, size_t bufsz) {
	if (!buffer) {
		return -1;
	}

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

// CEV: forward declaration
static void __attribute__((__noinline__)) chist_log_impl(
	const char *file,
	int line,
	enum log_level_t lvl,
	const char *format,
	...
) __attribute__((__format__(__printf__, 4, 5)));

// TODO: log to file
static void chist_log_impl(const char *file, int line, enum log_level_t lvl, const char *format, ...) {
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
	const char *level = chist_log_lvl_str(lvl);
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

#define unlikely(x) __builtin_expect(!!(x), 0)

#define chist_log(level, format, ...)                                          \
	do {                                                                       \
		if (unlikely(level >= log_level)) {                                    \
			chist_log_impl(__FILE__, __LINE__, level, format,  ##__VA_ARGS__); \
		}                                                                      \
	} while (0);

// TODO: remove unused macros
#define chist_debug(format, ...)  chist_log(LOG_LEVEL_DEBUG, format,  ##__VA_ARGS__)
#define chist_warn(format, ...)  chist_log(LOG_LEVEL_WARN, format,  ##__VA_ARGS__)
#define chist_error(format, ...) chist_log(LOG_LEVEL_ERROR, format,  ##__VA_ARGS__)

#define chist_fatal(format, ...)                            \
	do {                                                    \
		chist_log(LOG_LEVEL_FATAL, format,  ##__VA_ARGS__); \
		exit(EXIT_FAILURE);                                 \
	} while (0)

static char _chist_username[1024];

// TODO: don't allocate here!!!
static const char *get_current_user() {
	errno = 0;
	struct passwd *pw = getpwuid(getuid());
	assert(pw && pw->pw_name);
	if (pw && pw->pw_name) {
		return strncpy(_chist_username, pw->pw_name, sizeof(_chist_username));
	}
	chist_warn("getpwuid failed: %s", strerror(errno));
	return "UNKNOWN";
}

static char _chist_wd[4096];

static const char *get_working_directory() {
	const char *cwd = getcwd(_chist_wd, sizeof(_chist_wd));
	assert(cwd);
	if (cwd) {
		return cwd;
	}
	chist_warn("getcwd failed: %s", strerror(errno));
	return "UNKNOWN";
}


static long long parse_int_arg(const char *s, const char *arg_name) {
	char *endp;
	long long n = strtoll(s, &endp, 10);
	if ((n == 0 && errno != 0) || *endp) {
		const char *msg;
		if (errno != 0) {
			msg = strerror(errno);
		} else if (*endp) {
			msg = "invalid integer";
		} else {
			msg = "unknown error";
		}
		chist_fatal("error: parsing '%s' argument (%s): %s\n", arg_name, s, msg);
		assert(0);
		exit(EXIT_FAILURE); // unreachable
	}
	return n;
}

int main(int argc, char *argv[]) {
	// WARN
	const char * const server_socket = "/Users/cvieth/.local/share/histdb/socket/sock.sock";

	// session variables

	const char *user = get_current_user();

	pid_t ppid = getppid();

	const char *cwd = get_working_directory();

	char *session = NULL;
	bool status_set = false;
	long long status;

	// TODO: use this message for opt errors
	// 	gzip: unrecognized option '--foobar'

	int ch;
	int opt_index = 0;
	while ((ch = getopt_long(argc, argv, "dhs:c:", longopts, &opt_index)) != -1) {
		switch (ch) {
		case 'd':
			// TODO: add debug info using log + log level
			log_level = LOG_LEVEL_DEBUG;
			break;
		case 'h':
			usage(stdout);
			return 0;
		case 's':
			if (!optarg) {
				chist_fatal("error: missing 'session' argument: %s", longopts[opt_index].name);
			}
			session = strndup(optarg, 512);
			break;
		case 'c':
			if (!optarg) {
				chist_fatal("error: missing 'status-code' argument");
			}
			status = parse_int_arg(optarg, "status-code");
			status_set = true;
			break;
		default:
			chist_fatal("error: invalid argument: %s\n", argv[optind - 1]);
		}
	}
	argc -= optind;
	argv += optind;

	if (session == NULL) {
		chist_fatal("error: missing required argument: 'session'\n");
	}
	if (!status_set) {
		chist_fatal("error: missing required argument: 'status-code'\n");
	}
	if (argc < 1) {
		chist_fatal("error: missing COMMAND\n");
	}

	long long history_id = parse_int_arg(argv[0], "history_id");
	argc -= 1;
	argv += 1;

	chist_debug("options: session: '%s' status_code: '%lli' history_id: '%lli'",
		session, status, history_id);

	json_t *obj = json_object();
	if (!obj) {
		chist_fatal("fatal: failed to allocate JSON object\n");
	}
	if (json_object_set_new_nocheck(obj, "session_id", json_string(session)) != 0) {
		chist_fatal("error: setting: %s\n", "session_id");
	}
	if (json_object_set_new_nocheck(obj, "wd", json_string(cwd)) != 0) {
		chist_fatal("error: setting: %s\n", "wd");
	}
	if (json_object_set_new_nocheck(obj, "username", json_string(user)) != 0) {
		chist_fatal("error: setting: %s\n", "username");
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
	chist_debug("request_data: %s", request_data);

	int sockfd = socket(PF_UNIX, SOCK_STREAM, 0);
	if (sockfd == -1) {
		chist_fatal("WAT");
	}

	struct sockaddr_un addr;
	addr.sun_family = AF_UNIX;

	int addr_size = sizeof(addr.sun_path);
	if (snprintf(addr.sun_path, addr_size, "%s", server_socket) >= addr_size) {
		chist_fatal("error: socket path exceeds sockaddr_un.sun_path: %s", server_socket);
	}

	if (connect(sockfd, (struct sockaddr *)&addr, sizeof(struct sockaddr_un)) < 0) {
		chist_fatal("error: connecting to socket: %s", strerror(errno));
	}

	if (write(sockfd, request_data, strlen(request_data)) < 0) {
		close(sockfd);
		chist_fatal("error: writing to socket: %s", strerror(errno));
	}
	close(sockfd);
	free(request_data);

	return 0;
}

// CURL
/*
	FILE *request_file = fmemopen(request_data, strlen(request_data) + 1, "rb");
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

	CURLcode res = curl_easy_perform(curl_handle);
	if (res != CURLM_OK) {
		chist_fatal("error: curl: %s", curl_easy_strerror(res));
	}
	curl_easy_cleanup(curl_handle);

	// FILE *response_file = open_memstream(char **__bufp, size_t *__sizep)

	printf("DONE!\n");
	return 0;
*/
