# gosortimports

Command gosortimports is a more aggressive version of
[goimports](https://pkg.go.dev/golang.org/x/tools/cmd/goimports)
that handles very badly grouped imports and adds [gofmt's](https://pkg.go.dev/cmd/gofmt)
simplify code option (-s). Otherwise it is the same as [goimports].

**Note:** Since gosortimports undoes any existing import groupings it is
recommended to run it with the `-local` flag so that it can properly group
local imports. At some point gosortimports will support automatically deducing
the value of the local flag.

## Installation

```sh
go install github.com/charlievieth/utils/gosortimports
```

## Examples

The two following examples shows how gosortimports more aggressively handles
oddly grouped imports compared to goimports.

```go
import (
	"io"

	"m/a"

	"math"

	"fmt"
	"m/b"
)

// formatted with: `goimports -local=m/`

import (
	"io"

	"m/a"

	"math"

	"fmt"

	"m/b"
)

// formatted with: `gosortimports -local=m/`

import (
	"fmt"
	"io"
	"math"

	"m/a"
	"m/b"
)
```

<!--
```go
package main

import (
	"m/a"

	"fmt"

	"m/b"
)

// formatted with `gosortimports -local=m`:

import (
	"fmt"

	"m/a"
	"m/b"
)
```

* This is unchanged with `goimports -local=m`

`gosortimports -w -local=m`
```go
package main

import (
	"fmt"

	"m/a"
	"m/b"
)

// formatted with `gosortimports -local=m`:

import (
	"fmt"

	"m/a"
	"m/b"
)
```
-->
