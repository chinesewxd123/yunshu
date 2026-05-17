package service

import (
	"testing"
	"time"
)

func TestCanonicalAlertsFromAlertmanagerPayload(t *testing.T) {
	t.Parallel()
	p := AlertManagerPayload{
		Receiver: "platform-monitor",
		Status:   "firing",
		Alerts: []AlertManagerAlert{
			{
				Status:       "firing",
				Labels:       map[string]string{"alertname": "A"},
				Fingerprint:  "fp1",
				StartsAt:     time.Now(),
			},
			{
				Status:      "firing",
				Labels:      map[string]string{"alertname": "B"},
				Fingerprint: "fp2",
			},
		},
	}
	items := CanonicalAlertsFromAlertmanagerPayload(p)
	if len(items) != 2 {
		t.Fatalf("len=%d", len(items))
	}
	if items[0].Source != "platform_monitor" {
		t.Fatalf("source=%q", items[0].Source)
	}
	if items[0].Alert.Fingerprint != "fp1" {
		t.Fatalf("fp=%q", items[0].Alert.Fingerprint)
	}
}

func TestCanonicalAlertsCloudExpirySource(t *testing.T) {
	t.Parallel()
	p := AlertManagerPayload{Receiver: "cloud-expiry", Alerts: []AlertManagerAlert{{}}}
	items := CanonicalAlertsFromAlertmanagerPayload(p)
	if len(items) != 1 || items[0].Source != "cloud_expiry" {
		t.Fatalf("got %+v", items)
	}
}

func TestCanonicalAlertsDefaultSource(t *testing.T) {
	t.Parallel()
	p := AlertManagerPayload{Receiver: "team-a", Alerts: []AlertManagerAlert{{}}}
	items := CanonicalAlertsFromAlertmanagerPayload(p)
	if items[0].Source != "alertmanager" {
		t.Fatalf("got %q", items[0].Source)
	}
}
