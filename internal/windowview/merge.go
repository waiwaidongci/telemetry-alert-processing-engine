package windowview

func Merge(w Window, tail []Point) Window { return Window{Points: append(w.Points, tail...)} }
