package omhttp_test

import (
	"time"
)

var mockTime = time.Unix(1515151515, 757575757)

func mockNow() time.Time { return mockTime }
