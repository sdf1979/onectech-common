package aggregator

import "fmt"

type duration int64

func (d duration) String() string {
	return fmt.Sprintf("%.0f", float64(d)/1_000_000)
}

type durationAvg struct {
	duration int64
	count    int64
}

func (d durationAvg) String() string {
	if d.count != 0 {
		return fmt.Sprintf("%.3f", (float64(d.duration)/float64(d.count))/1_000_000)
	}
	return fmt.Sprintf("%.3f", float64(0))
}

type float64Prec2 float64

func (d float64Prec2) String() string {
	return fmt.Sprintf("%.2f", d)
}

type memoryMb int64

func (m memoryMb) String() string {
	return fmt.Sprintf("%.2f", float64(m)/1_048_576)
}
