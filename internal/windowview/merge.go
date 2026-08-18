package windowview

func Merge(w Window, tail []Point) Window {
	out := make([]Point, 0, len(w.Points)+len(tail))
	out = append(out, w.Points...)
	out = append(out, tail...)
	return Window{Points: out}
}
