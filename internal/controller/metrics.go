package controller

import (
	"math"
	"time"

	"github.com/A1exander-liU/comp-8005-assign-4/internal/shared"
)

// MetricItem is for recording global metric items.
type MetricItem int

const (
	MetricParseStart MetricItem = iota
	MetricParseEnd
	MetricCrackStart
	MetricCrackEnd
)

type JobMetric struct {
	assignmentStart, assignmentEnd time.Time
	returnStart, returnEnd         time.Time
	dispatchTime, crackTime        time.Duration
}

type HeartbeatMetric struct {
	heartbeatSeconds         int
	totalTested, deltaTested int
	rate                     float64
}

type Metric struct {
	globalTimings     map[MetricItem]time.Time
	jobTimings        map[int]*JobMetric
	checkpointTimings [][]time.Time
	heartbeatMetrics  []HeartbeatMetric
}

func NewMetric() *Metric {
	return &Metric{
		globalTimings: map[MetricItem]time.Time{},
		jobTimings:    map[int]*JobMetric{},

		heartbeatMetrics: []HeartbeatMetric{},
	}
}

// SetMetric sets the time for a metric item, overwriting the previous value if it's already set.
//
// Passing the empty value time.Time{} will use time.Now() rather than the
// provided time value.
func (m *Metric) SetMetric(i MetricItem, t time.Time) {
	if t.Equal(time.Time{}) {
		m.globalTimings[i] = time.Now()
	} else {
		m.globalTimings[i] = t
	}
}

// GetMetric gets the time value for a metric item.
//
// It will return false if the metric was not set already.
func (m *Metric) GetMetric(i MetricItem) (time.Time, bool) {
	t, ok := m.globalTimings[i]
	return t, ok
}

// GetJobMetric gets the job metric item for a given chunk.
//
// It will return false if no metrics for the chunk was set.
func (m *Metric) GetJobMetric(chunkID int) (*JobMetric, bool) {
	jm, ok := m.jobTimings[chunkID]
	return jm, ok
}

// SetJobMetric sets the time values for a given chunk.
//
// The passed job metric should set all the values it wishes to update. A time.time{}
// will not update the specifc time of the job metric.
func (m *Metric) SetJobMetric(chunkID int, jm JobMetric) {
	if j, ok := m.GetJobMetric(chunkID); !ok {
		m.jobTimings[chunkID] = &jm
	} else {
		if !jm.assignmentStart.Equal(time.Time{}) {
			j.assignmentStart = jm.assignmentStart
		}
		if !jm.assignmentEnd.Equal(time.Time{}) {
			j.assignmentEnd = jm.assignmentEnd
		}

		if !jm.returnStart.Equal(time.Time{}) {
			j.returnStart = jm.returnStart
		}
		if !jm.returnEnd.Equal(time.Time{}) {
			j.returnEnd = jm.returnEnd
		}

		if jm.dispatchTime != 0 {
			j.dispatchTime = jm.dispatchTime
		}

		if jm.crackTime != 0 {
			j.crackTime = jm.crackTime
		}
	}
}

// AddHeartbeatMetric adds a heartbeat metric for a given payload and heartbeat seconds.
func (m *Metric) AddHeartbeatMetric(payload shared.PayloadHearbeat, heartbeatSeconds int) {
	m.heartbeatMetrics = append(m.heartbeatMetrics, HeartbeatMetric{
		heartbeatSeconds: heartbeatSeconds,
		totalTested:      payload.TotalTested,
		deltaTested:      payload.DeltaTested,
		rate:             float64(payload.DeltaTested) / float64(heartbeatSeconds),
	})
}

// AddCheckpointTiming adds a checkpoint timing for a given start and end time.
func (m *Metric) AddCheckpointTiming(start, end time.Time) {
	m.checkpointTimings = append(m.checkpointTimings, []time.Time{start, end})
}

// HeartbeatMetrics calculates and returns the average rate and variance
// across all jobs.
func (m *Metric) HeartbeatMetrics() (float64, float64) {
	var jobRate float64

	for _, hm := range m.heartbeatMetrics {
		jobRate += hm.rate
	}

	averageRate := jobRate / float64(len(m.heartbeatMetrics))

	var jobRateSquareDiffs float64
	for _, hm := range m.heartbeatMetrics {
		jobRateSquareDiffs += math.Pow(float64(hm.rate)-float64(averageRate), 2)
	}

	variance := jobRateSquareDiffs / float64((len(m.heartbeatMetrics) - 1))

	return averageRate, variance
}

// EndToEndMetrics calculates and returns the end-to-end metrics for the cracking process.
//
// This includes the parse time, crack time, assignment overhead, dispatch overhead, return overhead, and checkpoint overhead.
func (m *Metric) EndToEndMetrics() (
	time.Duration,
	time.Duration,
	time.Duration,
	time.Duration,
	time.Duration,
	time.Duration,
) {
	parseTime := m.globalTimings[MetricParseEnd].Sub(m.globalTimings[MetricParseStart])
	crackTime := m.globalTimings[MetricCrackEnd].Sub(m.globalTimings[MetricCrackStart])

	var assignmentOverhead time.Duration
	var dispatchOverhead time.Duration
	var returnOverhead time.Duration

	for _, jm := range m.jobTimings {
		assignmentTime := jm.assignmentEnd.Sub(jm.assignmentStart)
		returnTime := jm.returnEnd.Sub(jm.returnStart)

		assignmentOverhead += assignmentTime
		dispatchOverhead += jm.dispatchTime
		returnOverhead += returnTime
	}

	var checkpointOverhead time.Duration
	for _, checkpoint := range m.checkpointTimings {
		start, end := checkpoint[0], checkpoint[1]
		checkpointTime := end.Sub(start)

		checkpointOverhead += checkpointTime
	}

	return parseTime, crackTime, assignmentOverhead, dispatchOverhead, returnOverhead, checkpointOverhead
}
