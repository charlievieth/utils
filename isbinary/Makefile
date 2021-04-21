.PHONY: build
build:
	go build --tags "libsqlite3 darwin"

.PHONY: install
install:
	go install --tags "libsqlite3 darwin"

.PHONY: test
test:
	go test --tags "libsqlite3 darwin"

.PHONY: clean
clean:
	rm *.test

.PHONY: all
all: test build
