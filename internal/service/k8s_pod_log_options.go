package service

import (
	"strings"
	"time"

	"yunshu/internal/pkg/constants"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func buildPodLogOptions(query PodLogsQuery, follow bool) (*corev1.PodLogOptions, error) {
	opts := &corev1.PodLogOptions{
		Container:  strings.TrimSpace(query.Container),
		Follow:     follow,
		Previous:   query.Previous,
		Timestamps: query.Timestamps,
	}
	if query.SinceSeconds > 0 {
		s := query.SinceSeconds
		opts.SinceSeconds = &s
	}
	if st := strings.TrimSpace(query.SinceTime); st != "" {
		t, err := time.Parse(time.RFC3339, st)
		if err != nil {
			t, err = time.ParseInLocation("2006-01-02 15:04:05", st, time.Local)
		}
		if err != nil {
			return nil, constants.ErrBadRequestWithMsg("since_time 格式无效，请使用 RFC3339 或 2006-01-02 15:04:05")
		}
		meta := metav1.NewTime(t)
		opts.SinceTime = &meta
	}
	if query.TailLines > 0 {
		tail := query.TailLines
		opts.TailLines = &tail
	} else if !follow && query.SinceSeconds <= 0 && strings.TrimSpace(query.SinceTime) == "" {
		tail := int64(300)
		opts.TailLines = &tail
	}
	if follow && opts.TailLines == nil {
		tail := int64(100)
		opts.TailLines = &tail
	}
	return opts, nil
}
