package k8sutil

import "testing"

func TestValidatePodContainerPath(t *testing.T) {
	if err := ValidatePodContainerPath("/var/log/app.log"); err != nil {
		t.Fatal(err)
	}
	if err := ValidatePodContainerPath("../etc/passwd"); err == nil {
		t.Fatal("expected reject ..")
	}
	if err := ValidatePodContainerPath("relative"); err == nil {
		t.Fatal("expected reject relative path")
	}
}
