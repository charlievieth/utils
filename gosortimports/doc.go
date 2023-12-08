/*
Command [gosortimports] is a more aggressive version of [goimports] that handles
very badly grouped imports and adds [gofmt's] simplify code option (-s).
Otherwise it is the same as [goimports].

	$ go install github.com/charlievieth/utils/gosortimports

In addition to fixing imports, gosortimports also formats your code in the
same style as gofmt so it can be used as a replacement for your editor's
gofmt-on-save hook.

For emacs, make sure you have the latest go-mode.el:

	https://github.com/dominikh/go-mode.el

Then in your .emacs file:

	(setq gofmt-command "gosortimports")
	(add-hook 'before-save-hook 'gofmt-before-save)

For vim, set "gofmt_command" to "gosortimports":

	https://golang.org/change/39c724dd7f252
	https://golang.org/wiki/IDEsAndTextEditorPlugins
	etc

For other editors, you probably know what to do.

To exclude directories in your $GOPATH from being scanned for Go
files, gosortimports respects a configuration file at
$GOPATH/src/.goimportsignore which may contain blank lines, comment
lines (beginning with '#'), or lines naming a directory relative to
the configuration file to ignore when scanning. No globbing or regex
patterns are allowed. Use the "-v" verbose flag to verify it's
working and see what gosortimports is doing.

Happy hacking!

[gosortimports]: https://github.com/charlievieth/utils/tree/master/gosortimports
[goimports]: https://pkg.go.dev/golang.org/x/tools/cmd/goimports
[gofmt's]: https://pkg.go.dev/cmd/gofmt
*/
package main
