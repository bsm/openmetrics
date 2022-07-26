package openmetrics

import (
	"bufio"
	"io"
	"math"
	"strconv"
	"time"
)

type bufferedWriter struct {
	*bufio.Writer
	tmp []byte
}

func (w *bufferedWriter) Reset(ww io.Writer) {
	w.Writer.Reset(ww)
	w.tmp = w.tmp[:0]
}

func (w *bufferedWriter) WriteIntro(prefix, name, unit, value string, escape bool) (total int, err error) {
	var n int

	n, err = w.WriteString(prefix)
	total += n
	if err != nil {
		return
	}

	n, err = w.writeName(name, unit, "")
	total += n
	if err != nil {
		return
	}

	if err = w.WriteByte(' '); err != nil {
		return
	}
	total++

	if escape {
		n, err = w.writeEscaped(value)
	} else {
		n, err = w.WriteString(value)
	}
	total += n
	if err != nil {
		return
	}

	if err = w.WriteByte('\n'); err != nil {
		return
	}
	total++

	return
}

func (w *bufferedWriter) WritePoint(name, unit string, lns, lvs []string, pt *MetricPoint) (total int, err error) {
	var n int

	n, err = w.writeName(name, unit, pt.Suffix.String())
	total += n
	if err != nil {
		return
	}

	n, err = w.writeLabels(lns, lvs, pt.Label)
	total += n
	if err != nil {
		return
	}

	n, err = w.writeValue(pt.Value, time.Time{}, pt.Suffix == SuffixCreated)
	total += n
	if err != nil {
		return
	}

	if pt.Exemplar != nil {
		n, err = w.WriteString(" # ")
		total += n
		if err != nil {
			return
		}

		n, err = w.writeExemplarLabels(pt.Exemplar.Labels)
		total += n
		if err != nil {
			return
		}

		n, err = w.writeValue(pt.Exemplar.Value, pt.Exemplar.Timestamp, false)
		total += n
		if err != nil {
			return
		}
	}

	if err = w.WriteByte('\n'); err != nil {
		return
	}
	total++

	return
}

func (w *bufferedWriter) writeName(name, unit, suffix string) (total int, err error) {
	var n int

	n, err = w.WriteString(name)
	total += n
	if err != nil {
		return
	}

	if unit != "" {
		if err = w.WriteByte('_'); err != nil {
			return
		}
		total++

		n, err = w.WriteString(unit)
		total += n
		if err != nil {
			return
		}
	}

	if suffix != "" {
		n, err = w.WriteString(suffix)
		total += n
		if err != nil {
			return
		}
	}

	return
}

func (w *bufferedWriter) writeEscaped(s string) (total int, err error) {
	var n int

	for _, r := range s {
		switch r {
		case '\n':
			n, err = w.WriteString(`\n`)
		case '"':
			n, err = w.WriteString(`\"`)
		case '\\':
			n, err = w.WriteString(`\\`)
		default:
			n, err = w.WriteRune(r)
		}

		total += n
		if err != nil {
			return
		}
	}
	return
}

func (w *bufferedWriter) writeExemplarLabels(set LabelSet) (total int, err error) {
	var n int
	if err = w.WriteByte('{'); err != nil {
		return
	}
	total++

	first := true
	for _, label := range set {
		if !label.IsZero() {
			n, err = w.writeLabel(label.Name, label.Value, first)
			total += n
			if err != nil {
				return
			}
			first = false
		}
	}

	if err = w.WriteByte('}'); err != nil {
		return
	}
	total++

	return
}

func (w *bufferedWriter) writeLabels(lns, lvs []string, extra Label) (total int, err error) {
	blank := extra.IsZero()
	if blank {
		for _, val := range lvs {
			if val != "" {
				blank = false
				break
			}
		}
	}
	if blank {
		return
	}

	var n int
	if err = w.WriteByte('{'); err != nil {
		return
	}
	total++

	first := true
	for i, name := range lns {
		if value := lvs[i]; value != "" {
			n, err = w.writeLabel(name, value, first)
			total += n
			if err != nil {
				return
			}
			first = false
		}
	}

	if !extra.IsZero() {
		n, err = w.writeLabel(extra.Name, extra.Value, first)
		total += n
		if err != nil {
			return
		}
	}

	if err = w.WriteByte('}'); err != nil {
		return
	}
	total++

	return
}

func (w *bufferedWriter) writeLabel(name, value string, first bool) (total int, err error) {
	var n int

	if !first {
		if err = w.WriteByte(','); err != nil {
			return
		}
		total++
	}

	n, err = w.WriteString(name)
	total += n
	if err != nil {
		return
	}

	n, err = w.WriteString(`="`)
	total += n
	if err != nil {
		return
	}

	n, err = w.writeEscaped(value)
	total += n
	if err != nil {
		return
	}

	if err = w.WriteByte('"'); err != nil {
		return
	}
	total++

	return
}

func (w *bufferedWriter) writeValue(value float64, timestamp time.Time, isEpoch bool) (total int, err error) {
	var n int

	if err = w.WriteByte(' '); err != nil {
		return
	}
	total++

	if isEpoch {
		n, err = w.writeEpoch(value)
	} else {
		n, err = w.writeFloat(value, 'g', -1)
	}
	total += n
	if err != nil {
		return
	}

	if !timestamp.IsZero() {
		if err = w.WriteByte(' '); err != nil {
			return
		}
		total++

		n, err = w.writeEpoch(asEpoch(timestamp))
		total += n
		if err != nil {
			return
		}
	}

	return
}

func (w *bufferedWriter) writeEpoch(e float64) (total int, err error) {
	if n := int(e*1e9) % 1e9; n == 0 {
		return w.writeInt(int64(e))
	} else if n%1e6 == 0 {
		return w.writeFloat(e, 'f', 3)
	}
	return w.writeFloat(e, 'f', 6)
}

func (w *bufferedWriter) writeFloat(val float64, format byte, precision int) (int, error) {
	val = math.Round(val*1e12) / 1e12
	w.tmp = strconv.AppendFloat(w.tmp[:0], val, format, precision, 64)
	return w.Write(w.tmp)
}

func (w *bufferedWriter) writeInt(val int64) (int, error) {
	w.tmp = strconv.AppendInt(w.tmp[:0], val, 10)
	return w.Write(w.tmp)
}
