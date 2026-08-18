package windowview

func owned(points []Point) []Point {
	cp := make([]Point, len(points))
	copy(cp, points)
	return cp
}
