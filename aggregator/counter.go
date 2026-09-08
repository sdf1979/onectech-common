package aggregator

import "fmt"

type Counter struct {
	Count int64
}

func (c *Counter) Apply(ctx UpdateContext) {
	c.Count += ctx.Count
}

func (c *Counter) Clone() Aggregator {
	return &Counter{Count: 0}
}

func (c *Counter) Score() int64 {
	return c.Count
}

func (c *Counter) String() string {
	return fmt.Sprintf("count=%d", c.Count)
}

func (C *Counter) Calculate(parent Aggregator) {}

type DurationCounter struct {
	Count           int64           `json:"Количество"`
	Duration        duration        `json:"Длительность (сек)"`
	DurationAvg     durationAvg     `json:"Длительность средняя (сек)"`
	DurationPercent durationPercent `json:"Длительность (% от родителя)"`
}

func (c *DurationCounter) Apply(ctx UpdateContext) {
	c.Count += ctx.Count
	c.Duration += duration(ctx.Duration)
	c.DurationAvg.count += ctx.Count
	c.DurationAvg.duration += ctx.Duration
}

func (c *DurationCounter) Clone() Aggregator {
	return &DurationCounter{Count: 0, Duration: 0, DurationAvg: durationAvg{0, 0}}
}

func (c *DurationCounter) Score() int64 {
	return int64(c.Duration)
}

func (c *DurationCounter) String() string {
	return fmt.Sprintf("Общая длительность: %.2f сек., Количество: %d, Средняя длительность: %.2f сек.",
		float64(c.Duration)/10000000, c.Count, (float64(c.Duration)/float64(c.Count))/1000000)
}

func (c *DurationCounter) Calculate(parent Aggregator) {
	if parent == nil {
		c.DurationPercent = 100.0
		return
	}
	p, ok := parent.(*DurationCounter)
	if !ok {
		c.DurationPercent = 0.0
		return
	}
	c.DurationPercent = 100.0 * durationPercent(c.Duration) / durationPercent(p.Duration)
}

type CallCounter struct {
	Count           int64           `json:"Количество"`
	Duration        duration        `json:"Длительность (сек)"`
	DurationAvg     durationAvg     `json:"Длительность средняя (сек)"`
	DurationPercent durationPercent `json:"Длительность (% от родителя)"`
	MemoryPeak      int64           `json:"Памяти выделено (байт)"`
	CpuTime         duration        `json:"Время процессора (сек)"`
}

func (c *CallCounter) Apply(ctx UpdateContext) {
	c.Count += ctx.Count
	c.Duration += duration(ctx.Duration)
	c.DurationAvg.count += ctx.Count
	c.DurationAvg.duration += ctx.Duration
	c.MemoryPeak += ctx.MemoryPeak
	c.CpuTime += duration(ctx.CpuTime)
}

func (c *CallCounter) Clone() Aggregator {
	return &CallCounter{Count: 0, Duration: 0, DurationAvg: durationAvg{0, 0}, DurationPercent: 0.0, MemoryPeak: 0, CpuTime: 0}
}

func (c *CallCounter) Score() int64 {
	return int64(c.Duration)
}

func (c *CallCounter) String() string {
	return fmt.Sprintf("Общая длительность: %.2f сек., Количество: %d, Средняя длительность: %.2f сек.",
		float64(c.Duration)/10000000, c.Count, (float64(c.Duration)/float64(c.Count))/1000000)
}

func (c *CallCounter) Calculate(parent Aggregator) {
	if parent == nil {
		c.DurationPercent = 100.0
		return
	}
	p, ok := parent.(*CallCounter)
	if !ok {
		c.DurationPercent = 0.0
		return
	}
	c.DurationPercent = 100.0 * durationPercent(c.Duration) / durationPercent(p.Duration)
}
