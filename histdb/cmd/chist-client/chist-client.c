#include <assert.h>
#include <ctype.h>
#include <errno.h>
#include <stdarg.h>
#include <stdbool.h>
#include <stdio.h>
#include <stdlib.h>
#include <strings.h>
#include <time.h>

#include <getopt.h>
#include <pwd.h>

#include <sys/types.h> // pid_t
#include <unistd.h>    // getppid

#include <sys/param.h> // MAXPATHLEN
#include <sys/time.h>  // gettimeofday, timeval

#include <curl/curl.h>
#include <jansson.h>

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

static const char * chist_log_lvl_str(enum log_level_t lvl) {
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
	assert(buffer);
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
	} while (0)

// TODO: remove unused macros
#define chist_debug(format, ...)  chist_log(LOG_LEVEL_DEBUG, format,  ##__VA_ARGS__)
#define chist_warn(format, ...)  chist_log(LOG_LEVEL_WARN, format,  ##__VA_ARGS__)
#define chist_error(format, ...) chist_log(LOG_LEVEL_ERROR, format,  ##__VA_ARGS__)

#define chist_fatal(format, ...)                            \
	do {                                                    \
		chist_log(LOG_LEVEL_FATAL, format,  ##__VA_ARGS__); \
		exit(EXIT_FAILURE);                                 \
	} while (0)

static const char *get_current_user() {
	errno = 0;
	struct passwd *pw = getpwuid(getuid());
	if (pw && pw->pw_name) {
		return pw->pw_name;
	}
	chist_warn("getpwuid failed: %s", strerror(errno));
	return "UNKNOWN";
}

static char _chist_wd[4096];

static const char *get_working_directory() {
	const char *cwd = getcwd(_chist_wd, sizeof(_chist_wd));
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
		exit(EXIT_FAILURE); // unreachable
	}
	return n;
}

struct chist_options {
	char             *log_file;
	enum log_level_t log_level;
};

struct chist_history_request {
	long long session_id;
	long long ppid;
	long long status_code;
	long long history_id;
	const char *wd;
	const char *username;
	const char **argv;
	int argc;
};

json_t *chist_json(const struct chist_history_request *req) {
	json_t *obj = json_object();
	if (!obj) {
		chist_fatal("fatal: failed to allocate JSON object\n");
	}
	if (json_object_set_new_nocheck(obj, "session_id", json_integer(req->session_id)) != 0) {
		chist_fatal("error: setting: session_id\n");
	}
	if (json_object_set_new_nocheck(obj, "wd", json_string(req->wd)) != 0) {
		chist_fatal("error: setting: wd\n");
	}
	if (json_object_set_new_nocheck(obj, "username", json_string(req->username)) != 0) {
		chist_fatal("error: setting: username\n");
	}
	if (json_object_set_new_nocheck(obj, "ppid", json_integer(req->ppid)) != 0) {
		chist_fatal("error: setting: ppid\n");
	}
	if (json_object_set_new_nocheck(obj, "status_code", json_integer(req->status_code)) != 0) {
		chist_fatal("error: setting: status_code\n");
	}
	if (json_object_set_new_nocheck(obj, "history_id", json_integer(req->history_id)) != 0) {
		chist_fatal("error: setting: history_id\n");
	}

	json_t *args = json_array();
	if (!args) {
		chist_fatal("fatal: failed to allocate JSON array\n");
	}
	for (int i = 0; i < req->argc; i++) {
		if (json_array_append_new(args, json_string(req->argv[i])) != 0) {
			chist_fatal("error: appending to array\n");
		}
	}
	if (json_object_set_new_nocheck(obj, "command", args) != 0) {
		chist_fatal("error: setting: %s\n", "command");
	}
	return obj;
}

struct memory_buffer {
	char   *data;
	size_t size;
};

static size_t write_memory_callback(void *contents, size_t size, size_t nmemb, void *userp) {
	size_t realsize = size * nmemb;
	struct memory_buffer *mem = (struct memory_buffer *)userp;

	char *ptr = realloc(mem->data, mem->size + realsize + 1);
	if (!ptr) {
		chist_error("not enough memory (realloc returned NULL)\n");
		return 0;
	}

	mem->data = ptr;
	memcpy(&(mem->data[mem->size]), contents, realsize);
	mem->size += realsize;
	mem->data[mem->size] = 0;

	return realsize;
}

static int chist_curl(const char *socket_path, const char *msg) {
	int exit_code = 0;

	CURL *curl = curl_easy_init();
	if (!curl) {
		chist_error("curl_easy_init: faild");
		goto error;
	}

	struct curl_slist hdrs = {
		.data = "Content-Type: application/json",
		.next = NULL,
	};

	struct memory_buffer chunk = {
		.data = NULL,
		.size = 0,
	};

	#define curl_setopt(option, param)                            \
		do {                                                      \
			CURLcode ret = curl_easy_setopt(curl, option, param); \
			if (unlikely(ret != CURLE_OK)) {                      \
				chist_error("curl_easy_setopt: " #option ": %s",  \
					curl_easy_strerror(ret));                     \
				goto error;                                       \
			}                                                     \
		} while (0)

	curl_setopt(CURLOPT_UNIX_SOCKET_PATH, socket_path);
	curl_setopt(CURLOPT_URL, "http://localhost/reflect");
	curl_setopt(CURLOPT_POSTFIELDS, msg);
	curl_setopt(CURLOPT_POSTFIELDSIZE, (long)strlen(msg));
	curl_setopt(CURLOPT_HTTPHEADER, &hdrs);

	// WARN
	curl_setopt(CURLOPT_WRITEFUNCTION, write_memory_callback);
	curl_setopt(CURLOPT_WRITEDATA, &chunk);

	#undef curl_setopt

	CURLcode res = curl_easy_perform(curl);
	if (res != CURLE_OK) {
		chist_error("curl_easy_perform: failed: %s", curl_easy_strerror(res));
		goto error;
	}

	long http_code;
	curl_easy_getinfo(curl, CURLINFO_RESPONSE_CODE, &http_code);
	if (http_code != 200) {
		chist_error("curl_easy_perform: status code: %li: response: %s",
				http_code, chunk.data);
		goto error;
	}

cleanup:
	if (curl) {
		curl_easy_cleanup(curl);
	}
	if (chunk.data) {
		free(chunk.data);
	}
	return exit_code;

error:
	exit_code = 1;
	goto cleanup;
}

int main(int argc, char *argv[]) {

	// WARN
	const char * const server_socket = "/Users/cvieth/.local/share/histdb/socket/sock.sock";

	// session variables

	const char *user = get_current_user();

	pid_t ppid = getppid();

	const char *cwd = get_working_directory();

	// WARN: make session an int !!!
	char *session = NULL;
	bool status_set = false;
	long long status;

	// TODO: use this message for opt errors
	// 	gzip: unrecognized option '--foobar'

	// TODO: fix arg parsing to support long opts

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
			if (!optarg || strlen(optarg) == 0) {
				chist_fatal("error: empty 'session' argument");
			}
			session = strndup(optarg, 512);
			break;
		case 'c':
			status = parse_int_arg(optarg, "status-code");
			status_set = true;
			break;
		case 0:
			if (strcmp("session", longopts[opt_index].name) == 0) {
				if (!optarg || strlen(optarg) == 0) {
					chist_fatal("error: empty 'session' argument");
				}
				session = strndup(optarg, 512);
				break;
			}
			if (strcmp("status-code", longopts[opt_index].name) == 0) {
				status = parse_int_arg(optarg, "status-code");
				status_set = true;
				break;
			}
			chist_fatal("option %s does not take a value\n", longopts[opt_index].name);
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
		chist_fatal("error: not enough arguments\n");
	}

	long long history_id = parse_int_arg(argv[0], "history_id");
	argc -= 1;
	argv += 1;

	chist_debug("options: session: '%s' status_code: '%lli' history_id: '%lli'",
		session, status, history_id);

	json_t *obj = json_object();
	if (!obj) {
		chist_fatal("fatal: failed to allocate JSON object");
	}
	if (json_object_set_new_nocheck(obj, "session_id", json_string(session)) != 0) {
		chist_fatal("error: setting: session_id");
	}
	if (json_object_set_new_nocheck(obj, "wd", json_string(cwd)) != 0) {
		chist_fatal("error: setting: wd");
	}
	if (json_object_set_new_nocheck(obj, "username", json_string(user)) != 0) {
		chist_fatal("error: setting: username");
	}
	if (json_object_set_new_nocheck(obj, "ppid", json_integer(ppid)) != 0) {
		chist_fatal("error: setting: ppid");
	}
	if (json_object_set_new_nocheck(obj, "status_code", json_integer(status)) != 0) {
		chist_fatal("error: setting: status_code");
	}
	if (json_object_set_new_nocheck(obj, "history_id", json_integer(history_id)) != 0) {
		chist_fatal("error: setting: history_id");
	}
	json_t *args = json_array();
	if (!args) {
		chist_fatal("fatal: failed to allocate JSON array");
	}
	for (int i = 0; i < argc; i++) {
		if (json_array_append_new(args, json_string(argv[i])) != 0) {
			chist_fatal("error: appending to array");
		}
	}
	if (json_object_set_new_nocheck(obj, "command", args) != 0) {
		chist_fatal("error: setting: command");
	}

	// TODO: destroy JSON object
	char *request_data = json_dumps(obj, JSON_ENSURE_ASCII|JSON_COMPACT);
	if (!request_data) {
		chist_fatal("json_dumps failed");
	}
	chist_debug("request_data: %s", request_data);

	chist_curl(server_socket, request_data);

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
