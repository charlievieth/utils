# CC=fake-gcc
# Debug (-g creates the *.dSYM directory)
#
# CFLAGS=-O2 -std=c11 -g -flto
CFLAGS=-O1 -std=c11
CFLAGS+=-I/usr/local/Cellar/jansson/2.13.1/include -L/usr/local/Cellar/jansson/2.13.1/lib -ljansson
CFLAGS+=-lcurl
# CFLAGS += $(shell pkg-config --cflags json-c)
# LDFLAGS += $(shell pkg-config --libs json-c)

#
# WARNINGS: https://gcc.gnu.org/onlinedocs/gcc/Warning-Options.html
#
CFLAGS+=-Wall -Wextra -Wpedantic -pedantic-errors -Wshadow
CFLAGS+=-Wswitch-enum -Wcast-qual -Wpointer-arith
CFLAGS+=-Wstrict-overflow=5 -Wcast-align
CFLAGS+=-Wno-gnu-zero-variadic-macro-arguments
#
# Performance:
#
# CFLAGS+=-Winline -Wdisabled-optimization
#
# Memory sanitizer:
#
# CFLAGS+=-fsanitize=address
#
# UB sanitizer:
#
# CFLAGS+=-fsanitize=undefined
#
# Race detector:
#
# CFLAGS+=-fsanitize=thread
#
# Stack protector flags:
# CFLAGS+=-fstack-protector-strong -fstack-check -fstack-protector
#
OUT=chist-client
RM=rm -rfv

.PHONY: all
all: clean build

.PHONY: address
address: CFLAGS+=-fsanitize=address
address: run

.PHONY: race
race: CFLAGS+=-fsanitize=thread
race: run

# build the binary (we do this so that we generate debug symbols)
chist-client:
	@$(CC) $(LDFLAGS) $(CFLAGS) -o $(OUT) chist-client.c

# Note:
#   $(RM) *.o forces rebuild
#   $(RM) $(OUT).o forces rebuild of target
#
# @$(RM) $(OUT).o
.PHONY: build
build: chist-client

.PHONY: debug
debug: CFLAGS+=-DDEBUG -fsanitize=thread
debug: run clean

.PHONY: run
run: clean build
	./$(OUT)

.PHONY: clean
clean:
	$(RM) *.o *.dSYM $(OUT)

.PHONY: compile-commands
compile-commands:
	bear fake-gcc $(LDFLAGS) $(CFLAGS) -o $(OUT) chist-client.c
