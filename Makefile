.PHONY: escape-regex
escape-regex:
	@cd ./escape-regex && go install

.PHONY: format-json
format-json:
	@cd ./format-json && go install

.PHONY: frequency
frequency:
	@cd ./frequency && go install

.PHONY: ghopen
ghopen:
	@cd ./ghopen && go install

.PHONY: godu
godu:
	@cd ./godu && go install

.PHONY: gouniq
gouniq:
	@cd ./gouniq && go install

.PHONY: gowd
gowd:
	@cd ./gowd && go install

.PHONY: isbinary
isbinary:
	@cd ./isbinary && go install

.PHONY: linecount
linecount:
	@cd ./linecount && go install

.PHONY: gotimeit
gotimeit:
	@cd ./gotimeit && go install

.PHONY: subl-completion
subl-completion:
	@cd ./subl-completion && go install

.PHONY: watchman-completion
watchman-completion:
	@cd ./watchman-completion && go install

# WARN: this is mostly here to document how to do this manually
.PHONY: install-subl-completion
install-subl-completion: subl-completion
	COMP_INSTALL=1 subl-completion

.PHONY: timestamp
timestamp:
	@cd ./timestamp && go install

# pathutils

.PHONY: extname
extname:
	@cd ./pathutils/cmd/extname && go install

.PHONY: gobasename
gobasename:
	@cd ./pathutils/cmd/gobasename && go install

.PHONY: godirname
godirname:
	@cd ./pathutils/cmd/godirname && go install

.PHONY: gocalc
gocalc:
	@cd ./gocalc && go install

.PHONY: pathutils
pathutils: extname gobasename godirname

.PHONY: rgsort
rgsort:
	@cd ./rgsort && go install

.PHONY: jq-completion
jq-completion:
	@cd ./jq-completion && go install

# Install frequently used utilities
all: escape-regex format-json frequency ghopen godu gouniq gowd \
	isbinary linecount subl-completion timestamp pathutils \
	watchman-completion rgsort jq-completion gocalc
