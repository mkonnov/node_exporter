package metric

import (
	"github.com/alecthomas/kingpin/v2"
	"github.com/prometheus/client_golang/prometheus"
	"slices"
)

const timestampLabelName = "timestamp"

var isTimestampLabelDisabled = kingpin.Flag(
	"metrics.label.timestamp.disabled",
	"Defines if a 'timestamp' label should be removed from metrics"+
		" defined in custom collectors.",
).Default("false").Bool()

func NewLabelNames(labels ...string) []string {
	if *isTimestampLabelDisabled {
		labels = slices.DeleteFunc(labels, func(e string) bool {
			return e == "timestamp"
		})
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
