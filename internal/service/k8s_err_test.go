package service

import (
	"errors"
	"testing"

	"yunshu/internal/pkg/apperror"
	"yunshu/internal/pkg/constants"

	"gorm.io/gorm"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestK8sMapAPIError(t *testing.T) {
	t.Parallel()
	gr := schema.GroupResource{Group: "apps", Resource: "deployments"}
	cases := []struct {
		name string
		in   error
		want string // error_code, "" = passthrough
	}{
		{"nil", nil, ""},
		{"not found", apierrors.NewNotFound(gr, "x"), "10004"},
		{"forbidden", apierrors.NewForbidden(gr, "x", errors.New("denied")), "10003"},
		{"conflict", apierrors.NewConflict(gr, "x", errors.New("exists")), "10005"},
		{"unauthorized", apierrors.NewUnauthorized("no"), "26002"},
		{"bad request", apierrors.NewBadRequest("bad"), "11020"},
		{"generic", errors.New("network"), ""},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := k8sMapAPIError(tc.in)
			if tc.in == nil {
				if got != nil {
					t.Fatalf("expected nil, got %v", got)
				}
				return
			}
			if tc.want == "" {
				if got != tc.in {
					t.Fatalf("expected passthrough, got %v", got)
				}
				return
			}
			ae, ok := apperror.IsAppError(got)
			if !ok || ae.ErrorCode != tc.want {
				t.Fatalf("expected code %s, got %v", tc.want, got)
			}
		})
	}
}

func TestK8sFailPreservesAppError(t *testing.T) {
	t.Parallel()
	orig := constants.ErrForbidden
	got := k8sFail("k8s.test", "op", orig)
	ae, ok := apperror.IsAppError(got)
	if !ok || ae.ErrorCode != "10003" {
		t.Fatalf("expected forbidden preserved, got %v", got)
	}
}

func TestK8sRepoErrNotFound(t *testing.T) {
	t.Parallel()
	got := k8sRepoErr("k8s.test", "get", gorm.ErrRecordNotFound)
	ae, ok := apperror.IsAppError(got)
	if !ok || ae.ErrorCode != "10004" {
		t.Fatalf("expected not found, got %v", got)
	}
}

func TestK8sMapAPIErrorUnauthorizedString(t *testing.T) {
	t.Parallel()
	got := k8sMapAPIError(errors.New("Unauthorized"))
	ae, ok := apperror.IsAppError(got)
	if !ok || ae.ErrorCode != "26002" {
		t.Fatalf("expected 26002, got %v", got)
	}
}

func TestK8sFailMapsNotFound(t *testing.T) {
	t.Parallel()
	err := apierrors.NewNotFound(schema.GroupResource{Resource: "pods"}, "n1")
	got := k8sFail("k8s.workload", "get", err)
	ae, ok := apperror.IsAppError(got)
	if !ok || ae.ErrorCode != "10004" {
		t.Fatalf("expected 10004, got %v", got)
	}
}
