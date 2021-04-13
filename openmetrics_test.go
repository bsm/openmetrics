package openmetrics_test

import (
	"time"

	"github.com/bsm/openmetrics"
)

var (
	mockTime = time.Unix(1515151515, 757575757)
	mockDesc = openmetrics.Desc{Name: "mock"}
)

func mockNow() time.Time { return mockTime }
