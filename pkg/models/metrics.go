package models

import "time"

type MetricData struct {
	Service          string
	Environment      string
	NormalizedRoute  string
	Timestamp        time.Time
	RequestsTotal    int64
	RequestsGood     int64
	LatencySum       float64
	LatencyCount     int64
}

func (m *MetricData) AverageLatency() float64 {
	if m.LatencyCount == 0 {
		return 0
	}
	return m.LatencySum / float64(m.LatencyCount)
}

type MetricKey struct {
	Service         string
	Environment     string
	NormalizedRoute string
	Minute          time.Time
}

type MetricAggregator struct {
	RequestsTotal int64
	RequestsGood  int64
	LatencySum    float64
	LatencyCount  int64
}

func (a *MetricAggregator) Add(entry *ALBLogEntry) {
	a.RequestsTotal++
	if entry.IsSuccessful() {
		a.RequestsGood++
	}
	
	latency := entry.TotalLatency()
	if latency > 0 {
		a.LatencySum += latency
		a.LatencyCount++
	}
}