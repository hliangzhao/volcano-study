/*
Copyright 2021-2022 The Volcano Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package metrics

// fully checked and understood

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	jobShare = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Subsystem: VolcanoNamespace,
			Name:      "job_share",
			Help:      "Share for one job",
		},
		[]string{"job_ns", "job_id"},
	)

	jobRetryCount = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Subsystem: VolcanoNamespace,
			Name:      "job_retry_counts",
			Help:      "Number of retry counts for one job",
		},
		[]string{"job_id"},
	)
)

// UpdateJobShare records share for one job.
func UpdateJobShare(jobNS, jobID string, share float64) {
	jobShare.WithLabelValues(jobNS, jobID).Set(share)
}

// RegisterJobRetries gets total number of job retries.
func RegisterJobRetries(jobID string) {
	jobRetryCount.WithLabelValues(jobID).Inc()
}

// DeleteJobShare delete jobShare for one job.
func DeleteJobShare(jobNs, jobID string) {
	jobShare.DeleteLabelValues(jobNs, jobID)
}
