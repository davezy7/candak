package downloader

import (
	"time"
)

// MetricsCalculator tracks byte progress over time to compute transfer speed and ETA.
type MetricsCalculator struct {
	lastBytes int64
	lastTime  time.Time
	speedEMA  float64
	alpha     float64 // Smoothing factor (0.0 to 1.0)
}

// NewMetricsCalculator constructs a new calculator initialized at current time.
func NewMetricsCalculator(initialBytes int64) *MetricsCalculator {
	return &MetricsCalculator{
		lastBytes: initialBytes,
		lastTime:  time.Now(),
		alpha:     0.7, // 70% current instant speed, 30% historical EMA
	}
}

// Update calculates current transfer speed and remaining seconds (ETA).
func (m *MetricsCalculator) Update(now time.Time, currentBytes, totalBytes int64) (speed int64, eta int64) {
	deltaSec := now.Sub(m.lastTime).Seconds()
	if deltaSec <= 0 {
		return int64(m.speedEMA), 0
	}

	bytesDelta := currentBytes - m.lastBytes
	if bytesDelta < 0 {
		bytesDelta = 0
	}

	instantSpeed := float64(bytesDelta) / deltaSec
	if m.speedEMA == 0 {
		m.speedEMA = instantSpeed
	} else {
		m.speedEMA = (m.alpha * instantSpeed) + ((1.0 - m.alpha) * m.speedEMA)
	}

	m.lastBytes = currentBytes
	m.lastTime = now

	speed = int64(m.speedEMA)
	if speed > 0 && totalBytes > currentBytes {
		eta = (totalBytes - currentBytes) / speed
	} else {
		eta = 0
	}

	return speed, eta
}
