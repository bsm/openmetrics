package openmetrics_test

import (
	"reflect"
	"testing"

	. "github.com/bsm/openmetrics"
)

func TestInfo_AppendPoints(t *testing.T) {
	ist := NewInfo()
	got, err := ist.AppendPoints(nil, &mockDesc)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if exp := []MetricPoint{
		{Suffix: SuffixInfo, Value: 1},
	}; !reflect.DeepEqual(exp, got) {
		t.Fatalf("expected:\n\t%+v, got:\n\t%+v", exp, got)
	}
}

func BenchmarkInfo(b *testing.B) {
	ist := NewInfo()
	pts := []MetricPoint{}
	b.Run("AppendPoints", func(b *testing.B) {
		var err error
		for i := 0; i < b.N; i++ {
			if pts, err = ist.AppendPoints(pts[:0], &mockDesc); err != nil {
				b.Fatalf("expected no error, got %v", err)
			}
		}
	})
}
