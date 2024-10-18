package metric

import (
	"github.com/alecthomas/kingpin/v2"
	"github.com/prometheus/client_golang/prometheus"
)

const timestampLabelName = "timestamp"

var isTimestampLabelDisabled = kingpin.Flag(
	"metrics.label.timestamp.disabled",
	"Defines if a 'timestamp' label should be removed from metrics"+
		" defined in custom collectors.",
).Default("false").Bool()

func NewLabelNames(labels ...string) []string {
	shouldResize := false

	writeI := 0
	for _, label := range labels {
		if label != timestampLabelName {
			labels[writeI] = label
			writeI++
			continue
		}
		if *isTimestampLabelDisabled {
			shouldResize = true
		} else {
			break
		}
	}

	if shouldResize {
		labels = labels[:len(labels)-1]
	}

	return labels
}

func NewLabels(labels prometheus.Labels) prometheus.Labels {
	_, includesTimestamp := labels[timestampLabelName]
	if includesTimestamp && *isTimestampLabelDisabled {
		delete(labels, timestampLabelName)
	}
	return labels
}
