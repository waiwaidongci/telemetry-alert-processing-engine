package windowview

type Point struct{ Value int }
type Window struct{ Points []Point }

func New(points []Point) Window { return Window{Points: points} }
