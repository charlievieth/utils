.PHONY: build
build:
	go build --tags "libsqlite3 darwin"

.PHONY: install
install:
	CGO_CFLAGS="-g -O3 -march=native -mtune=native" go install --tags "libsqlite3 darwin"

.PHONY: test
test:
	go test --tags "libsqlite3 darwin"

.PHONY: clean
clean:
	rm *.test

.PHONY: all
all: test build
