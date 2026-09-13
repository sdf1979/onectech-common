package aggregator

import "fmt"

type Counter struct {
	Count int64 `json:"Количество"`
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
	Count           int64        `json:"Количество"`
	Duration        duration     `json:"Время (сек)"`
	DurationAvg     durationAvg  `json:"Время Avg (сек)"`
	DurationPercent float64Prec2 `json:"Время (% группы)"`
	Rows            int64        `json:"      Строк     "`
	RowsAvg         float64Prec2 `json:" Строк Avg "`
}

func (c *DurationCounter) Apply(ctx UpdateContext) {
	c.Count += ctx.Count
	c.Duration += duration(ctx.Duration)
	c.DurationAvg.count += ctx.Count
	c.DurationAvg.duration += ctx.Duration
	c.Rows += ctx.Rows
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
	if c.Count != 0 {
		c.RowsAvg = float64Prec2(c.Rows) / float64Prec2(c.Count)
	}
	if parent == nil {
		c.DurationPercent = 100.0
		return
	}
	p, ok := parent.(*DurationCounter)
	if !ok {
		c.DurationPercent = 0.0
		return
	}

	if p.Duration != 0 {
		c.DurationPercent = 100.0 * float64Prec2(c.Duration) / float64Prec2(p.Duration)
	}
}

type CallCounter struct {
	Count           int64        `json:"Кол-во"`
	Duration        duration     `json:"Время (сек)"`
	DurationAvg     durationAvg  `json:"Время Avg (сек)"`
	DurationPercent float64Prec2 `json:"Время (% группы)"`
	CpuTime         duration     `json:"Время CPU (сек)"`
	CpuTimeAvg      durationAvg  `json:"Время CPU Avg (сек)"`
	CpuTimePercent  float64Prec2 `json:"Время CPU (% группы)"`
	MemoryPeak      memoryMb     `json:"Память (Mb)"`
}

func (c *CallCounter) Apply(ctx UpdateContext) {
	c.Count += ctx.Count
	c.Duration += duration(ctx.Duration)
	c.DurationAvg.count += ctx.Count
	c.DurationAvg.duration += ctx.Duration
	c.MemoryPeak += memoryMb(ctx.MemoryPeak)
	c.CpuTime += duration(ctx.CpuTime)
	c.CpuTimeAvg.count += ctx.Count
	c.CpuTimeAvg.duration += ctx.Duration
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
		c.CpuTimePercent = 100.0
		return
	}
	p, ok := parent.(*CallCounter)
	if !ok {
		c.DurationPercent = 0.0
		c.CpuTimePercent = 0.0
		return
	}
	if p.Duration != 0 {
		c.DurationPercent = 100.0 * float64Prec2(c.Duration) / float64Prec2(p.Duration)
	}
	if p.CpuTime != 0 {
		c.CpuTimePercent = 100.0 * float64Prec2(c.CpuTime) / float64Prec2(p.CpuTime)
	}
}
