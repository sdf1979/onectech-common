package aggregator

import "fmt"

type duration int64

func (d duration) String() string {
	return fmt.Sprintf("%.3f", float64(d)/1_000_000)
}

type durationAvg struct {
	duration int64
	count    int64
}

func (d durationAvg) String() string {
	return fmt.Sprintf("%.3f", (float64(d.duration)/float64(d.count))/1_000_000)
}

type durationPercent float64

func (d durationPercent) String() string {
	return fmt.Sprintf("%.2f", d)
}
