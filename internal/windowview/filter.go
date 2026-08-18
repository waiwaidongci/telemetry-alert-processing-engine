package windowview

func Positive(points []Point) []Point {
	out := make([]Point, 0, len(points))
	for _, p := range points {
		if p.Value > 0 {
			out = append(out, p)
		}
	}
	return out
}
