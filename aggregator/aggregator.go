package aggregator

type UpdateContext struct {
	Count      int64
	Duration   int64
	MemoryPeak int64
	CpuTime    int64
}

type Aggregator interface {
	Apply(ctx UpdateContext)
	Clone() Aggregator
	Score() int64
	String() string
	Calculate(parent Aggregator)
}
