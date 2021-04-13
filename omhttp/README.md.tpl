# OpenMetrics HTTP

The `omhttp` package provides convenience helpers for HTTP servers.

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
