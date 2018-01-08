package v1

import (
	"testing"

	corev1 "k8s.io/api/core/v1"

	"github.com/oracle/mysql-operator/pkg/version"
)

func TestRestoreEnsureDefaults(t *testing.T) {
	// test a version is set if one does not exist.
	bv1 := version.GetBuildVersion()
	r := MySQLRestore{
		Spec: RestoreSpec{
			ClusterRef: &corev1.LocalObjectReference{
				Name: "foo",
			},
			BackupRef: &corev1.LocalObjectReference{
				Name: "foo",
			},
		},
	}
	dr := *r.EnsureDefaults()
	if GetOperatorVersionLabel(dr.Labels) != bv1 {
		t.Errorf("Expected restore version label: '%s'", bv1)
	}
	// test a version is not set if one already exists.
	bv2 := "test-existing-build-version"
	r2 := MySQLRestore{}
	r2.Labels = make(map[string]string)
	SetOperatorVersionLabel(r2.Labels, bv2)
	dr2 := *r2.EnsureDefaults()
	if GetOperatorVersionLabel(dr2.Labels) != bv2 {
		t.Errorf("Expected restore version label: '%s'", bv2)
	}
}

func TestRestoreValidate(t *testing.T) {
	// Test a malformed restore returns errors.
	r := MySQLRestore{
		Spec: RestoreSpec{
			ClusterRef: &corev1.LocalObjectReference{
				Name: "foo",
			},
			BackupRef: &corev1.LocalObjectReference{
				Name: "foo",
			},
		},
	}
	rErr := r.Validate()
	if rErr == nil {
		t.Error("Restore should have had a validation error.")
	}
	// Test a valid restore returns no errors.
	r.Labels = make(map[string]string)
	SetOperatorVersionLabel(r.Labels, "some-build-version")
	rErr = r.Validate()
	if rErr != nil {
		t.Errorf("Restore should have had no validation errors: %v", rErr)
	}
}
