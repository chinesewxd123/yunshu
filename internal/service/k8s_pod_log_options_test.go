package service

import (
	"testing"
)

func TestBuildPodLogOptionsPreviousSince(t *testing.T) {
	opts, err := buildPodLogOptions(PodLogsQuery{
		Previous:     true,
		SinceSeconds: 3600,
		Timestamps:   true,
		TailLines:    100,
	}, false)
	if err != nil {
		t.Fatal(err)
	}
	if !opts.Previous || !opts.Timestamps {
		t.Fatalf("expected previous and timestamps")
	}
	if opts.SinceSeconds == nil || *opts.SinceSeconds != 3600 {
		t.Fatalf("since_seconds")
	}
	if opts.TailLines == nil || *opts.TailLines != 100 {
		t.Fatalf("tail_lines")
	}
}

func TestBuildPodLogOptionsInvalidSinceTime(t *testing.T) {
	_, err := buildPodLogOptions(PodLogsQuery{SinceTime: "not-a-time"}, false)
	if err == nil {
		t.Fatal("expected error")
	}
}
