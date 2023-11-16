RUN_NUM_WRITERS_FLAG    ?= -n 128
RUN_WRITE_INTERVAL_FLAG ?= -d 1ns
RUN_SYNC_FLAGS          ?= -sync
RUN_EXTRA_FLAGS         ?=
RUN_FLAGS ?= $(RUN_NUM_WRITERS_FLAG) $(RUN_WRITE_INTERVAL_FLAG) $(RUN_SYNC_FLAGS)

.PHONY: all
all: build

.PHONY: build
build: clean
	@# need -buildvcs=false for building on a shared vmhgfs-fuse drive
	@go build -buildvcs=false

.PHONY: run
run: build
	@./run.bash $(RUN_FLAGS)

.PHONY: clean
clean:
	@go clean
