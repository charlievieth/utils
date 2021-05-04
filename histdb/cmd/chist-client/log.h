#ifndef CHIST_LOG_H
#define CHIST_LOG_H

#include <stdlib.h>

enum log_level_t {
	LOG_LEVEL_DEBUG,
	LOG_LEVEL_INFO,
	LOG_LEVEL_WARN,
	LOG_LEVEL_ERROR,
	LOG_LEVEL_FATAL
};

enum log_level_t log_level = LOG_LEVEL_ERROR;

const char *chist_log_level_string(enum log_level_t lvl)
	__attribute__((returns_nonnull));

int chist_log_level_parse(const char *str, enum log_level_t *lvl)
	__attribute__((__nonnull__));

void __attribute__((__noinline__)) chist_log_impl(
	const char *file,
	int line,
	enum log_level_t lvl,
	const char *format,
	...
) __attribute__((__format__(__printf__, 4, 5)));

#define chist_log(level, format, ...)                                          \
	do {                                                                       \
		if (__builtin_expect(level >= log_level, 0)) {                         \
			chist_log_impl(__FILE__, __LINE__, level, format,  ##__VA_ARGS__); \
		}                                                                      \
	} while (0)

// TODO: remove unused macros
#define chist_debug(format, ...)  chist_log(LOG_LEVEL_DEBUG, format,  ##__VA_ARGS__)
#define chist_warn(format, ...)   chist_log(LOG_LEVEL_WARN, format,  ##__VA_ARGS__)
#define chist_error(format, ...)  chist_log(LOG_LEVEL_ERROR, format,  ##__VA_ARGS__)

#define chist_fatal(format, ...)                            \
	do {                                                    \
		chist_log(LOG_LEVEL_FATAL, format,  ##__VA_ARGS__); \
		exit(EXIT_FAILURE);                                 \
	} while (0)


#endif /* CHIST_LOG_H */
