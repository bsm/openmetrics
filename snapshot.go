package openmetrics

type snapshot struct {
	desc Desc
	mt   MetricType
	pts  []MetricPoint
	lvs  [][]string
	off  []int
	cos  uint64Slice
}

func (s *snapshot) Reset(desc Desc, mt MetricType) {
	*s = snapshot{
		desc: desc,
		mt:   mt,
		pts:  s.pts[:0],
		lvs:  s.lvs[:0],
		off:  s.off[:0],
		cos:  s.cos[:0],
	}
}

func (s *snapshot) Append(m *metricWithLabels) (err error) {
	// append points, exit early if none collected
	origSize := len(s.pts)
	s.pts, err = m.met.AppendPoints(s.pts, &s.desc)
	if err != nil || len(s.pts) == origSize {
		return
	}

	s.off = append(s.off, len(s.pts))
	s.lvs = append(s.lvs, m.lvs)
	return
}

func (s *snapshot) WriteTo(bw *bufferedWriter) (total int64, err error) {
	if len(s.pts) == 0 {
		return
	}

	var n int
	n, err = s.desc.writeTo(bw, s.mt)
	total += int64(n)
	if err != nil {
		return
	}

	off := 0
	for i, max := range s.off {
		lvs := s.lvs[i]
		for ; off < max; off++ {
			pt := s.pts[off]
			n, err = bw.WritePoint(s.desc.Name, s.desc.Unit, s.desc.Labels, lvs, &pt)
			total += int64(n)
			if err != nil {
				return
			}
		}
	}

	return
}
