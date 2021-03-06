# OpenMetrics

[![Go Reference](https://pkg.go.dev/badge/github.com/bsm/openmetrics.svg)](https://pkg.go.dev/github.com/bsm/openmetrics)
[![Test](https://github.com/bsm/openmetrics/actions/workflows/test.yml/badge.svg)](https://github.com/bsm/openmetrics/actions/workflows/test.yml)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)

OpenMetrics is a standalone, dependency-free implementation of [OpenMetrics v1.0](https://github.com/OpenObservability/OpenMetrics/blob/main/specification/OpenMetrics.md) specification for [Go](https://golang.org/).

## Example

To expose metrics on a HTTP server endpoint and to instrument HTTP servers, please see examples in the [omhttp](./omhttp/) package.

```go
import(
	"bytes"
	"fmt"

	"github.com/bsm/openmetrics"
)

func main() {
	reg := openmetrics.NewConsistentRegistry(mockNow)	// or, openmetrics.DefaultRegistry()
	requestCount := reg.Counter(openmetrics.Desc{
		Name:	"http_request",
		Help:	"A counter example",
		Labels:	[]string{"status"},
	})
	responseTime := reg.Histogram(openmetrics.Desc{
		Name:	"http_request",
		Unit:	"seconds",
		Help:	"A histogram example",
		Labels:	[]string{"status"},
	}, .005, .01, .05, .1, .5, 1, 5, 10)

	requestCount.With("200").Add(1)
	responseTime.With("200").Observe(0.56)

	var buf bytes.Buffer
	if _, err := reg.WriteTo(&buf); err != nil {
		panic(err)
	}
	fmt.Print(buf.String())

}
```

## License

```text
Copyright 2021 Black Square Media Ltd

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
```
