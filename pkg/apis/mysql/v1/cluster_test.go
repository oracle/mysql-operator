package v1

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

func TestValidVersion(t *testing.T) {
	for _, version := range validVersions {
		errList := validateVersion(version, field.NewPath("spec", "version"))
		if len(errList) > 0 {
			t.Fail()
		}
	}
}

func TestInvalidVersion(t *testing.T) {
	err := validateVersion("1.2.3", field.NewPath("spec", "version"))
	if err == nil {
		t.Fail()
	}
}

func TestDefaultReplicas(t *testing.T) {
	cluster := &MySQLCluster{}
	cluster.EnsureDefaults()

	if cluster.Spec.Replicas != defaultReplicas {
		t.Errorf("Expected default replicas to be %d but got %d", defaultReplicas, cluster.Spec.Replicas)
	}
}

func TestDefaultVersion(t *testing.T) {
	cluster := &MySQLCluster{}
	cluster.EnsureDefaults()

	if cluster.Spec.Version != defaultVersion {
		t.Errorf("Expected default version to be %s but got %s", defaultVersion, cluster.Spec.Version)
	}
}

func TestRequiresConfigMount(t *testing.T) {
	cluster := &MySQLCluster{}
	cluster.EnsureDefaults()

	if cluster.RequiresConfigMount() {
		t.Errorf("Cluster without configRef should not require a config mount")
	}

	cluster = &MySQLCluster{
		Spec: MySQLClusterSpec{
			ConfigRef: &corev1.LocalObjectReference{
				Name: "customconfig",
			},
		},
	}

	if !cluster.RequiresConfigMount() {
		t.Errorf("Cluster with configRef should require a config mount")
	}
}
