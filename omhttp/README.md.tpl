# OpenMetrics HTTP

[![Go Reference](https://pkg.go.dev/badge/github.com/bsm/openmetrics.svg)](https://pkg.go.dev/github.com/bsm/openmetrics/omhttp)

The `omhttp` package provides HTTP-related convenience helpers.

## Examples

To expose metrics on an HTTP server endpoint:

```go
package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"

	"github.com/bsm/openmetrics"
	"github.com/bsm/openmetrics/omhttp"
)

func main() {{ "ExampleNewHandler" | code }}
```

To instrument HTTP servers:

```go
package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"time"

	"github.com/bsm/openmetrics"
	"github.com/bsm/openmetrics/omhttp"
)

func main() {{ "ExampleInstrument" | code }}
```
