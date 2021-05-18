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

type errorCollector struct{ errs []error }

func (c *errorCollector) OnError(err error) { c.errs = append(c.errs, err) }
func (c *errorCollector) Reset()            { c.errs = c.errs[:0] }
func (c *errorCollector) Errors() []string {
	if len(c.errs) == 0 {
		return nil
	}

	msgs := make([]string, 0, len(c.errs))
	for _, err := range c.errs {
		msgs = append(msgs, err.Error())
	}
	return msgs
}
