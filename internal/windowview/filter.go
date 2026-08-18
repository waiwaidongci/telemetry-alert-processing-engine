package windowview

func Positive(points []Point) []Point {
	out := points[:0]
	for _, p := range points {
		if p.Value > 0 {
			out = append(out, p)
		}
	}
	return out
}
