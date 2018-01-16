package labeler

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	fake "k8s.io/client-go/kubernetes/fake"

	constants "github.com/oracle/mysql-operator/pkg/constants"
)

func TestPrimarySelector(t *testing.T) {
	testCases := []struct {
		name    string
		labels  map[string]string
		matches int
	}{
		{
			name: "primary",
			labels: map[string]string{
				constants.MySQLClusterLabel:     "primary",
				constants.LabelMySQLClusterRole: constants.MySQLClusterRolePrimary,
			},
			matches: 1,
		}, {
			name: "secondary",
			labels: map[string]string{
				constants.MySQLClusterLabel:     "secondary",
				constants.LabelMySQLClusterRole: constants.MySQLClusterRoleSecondary,
			},
			matches: 0,
		}, {
			name: "blank",
			labels: map[string]string{
				constants.MySQLClusterLabel:     "blank",
				constants.LabelMySQLClusterRole: "",
			},
			matches: 0,
		}, {
			name: "missing",
			labels: map[string]string{
				constants.MySQLClusterLabel: "missing",
			},
			matches: 0,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			client := fake.NewSimpleClientset(newPodWithLabels(tt.name, tt.labels))
			pods, err := client.CoreV1().Pods(metav1.NamespaceDefault).List(metav1.ListOptions{
				LabelSelector: PrimarySelector(tt.name).String(),
			})
			if err != nil {
				t.Fatalf("Expected no error listing pods, got: %+v", err)
			}
			if len(pods.Items) != tt.matches {
				t.Errorf("Expected %d matches for Pod with labels %+v but got %d", tt.matches, tt.labels, len(pods.Items))
			}
		})
	}
}

func TestSecondarySelector(t *testing.T) {
	testCases := []struct {
		name    string
		labels  map[string]string
		matches int
	}{
		{
			name: "secondary",
			labels: map[string]string{
				constants.MySQLClusterLabel:     "secondary",
				constants.LabelMySQLClusterRole: constants.MySQLClusterRoleSecondary,
			},
			matches: 1,
		}, {
			name: "primary",
			labels: map[string]string{
				constants.MySQLClusterLabel:     "primary",
				constants.LabelMySQLClusterRole: constants.MySQLClusterRolePrimary,
			},
			matches: 0,
		}, {
			name: "blank",
			labels: map[string]string{
				constants.MySQLClusterLabel:     "blank",
				constants.LabelMySQLClusterRole: "",
			},
			matches: 0,
		}, {
			name: "missing",
			labels: map[string]string{
				constants.MySQLClusterLabel: "missing",
			},
			matches: 0,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			client := fake.NewSimpleClientset(newPodWithLabels(tt.name, tt.labels))
			pods, err := client.CoreV1().Pods(metav1.NamespaceDefault).List(metav1.ListOptions{
				LabelSelector: SecondarySelector(tt.name).String(),
			})
			if err != nil {
				t.Fatalf("Expected no error listing pods, got: %+v", err)
			}
			if len(pods.Items) != tt.matches {
				t.Errorf("Expected %d matches for Pod with labels %+v but got %d", tt.matches, tt.labels, len(pods.Items))
			}
		})
	}
}

func TestNonPrimarySelector(t *testing.T) {
	testCases := []struct {
		name    string
		labels  map[string]string
		matches int
	}{
		{
			name: "secondary",
			labels: map[string]string{
				constants.MySQLClusterLabel:     "secondary",
				constants.LabelMySQLClusterRole: constants.MySQLClusterRoleSecondary,
			},
			matches: 1,
		}, {
			name: "primary",
			labels: map[string]string{
				constants.MySQLClusterLabel:     "primary",
				constants.LabelMySQLClusterRole: constants.MySQLClusterRolePrimary,
			},
			matches: 0,
		}, {
			name: "blank",
			labels: map[string]string{
				constants.MySQLClusterLabel:     "blank",
				constants.LabelMySQLClusterRole: "",
			},
			matches: 1,
		}, {
			name: "missing",
			labels: map[string]string{
				constants.MySQLClusterLabel: "missing",
			},
			matches: 1,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			client := fake.NewSimpleClientset(newPodWithLabels(tt.name, tt.labels))
			pods, err := client.CoreV1().Pods(metav1.NamespaceDefault).List(metav1.ListOptions{
				LabelSelector: NonPrimarySelector(tt.name).String(),
			})
			if err != nil {
				t.Fatalf("Expected no error listing pods, got: %+v", err)
			}
			if len(pods.Items) != tt.matches {
				t.Errorf("Expected %d matches for Pod with labels %+v but got %d", tt.matches, tt.labels, len(pods.Items))
			}
		})
	}
}

func TestHasRoleSelector(t *testing.T) {
	testCases := []struct {
		name    string
		labels  map[string]string
		matches int
	}{
		{
			name: "secondary",
			labels: map[string]string{
				constants.MySQLClusterLabel:     "secondary",
				constants.LabelMySQLClusterRole: constants.MySQLClusterRoleSecondary,
			},
			matches: 1,
		}, {
			name: "primary",
			labels: map[string]string{
				constants.MySQLClusterLabel:     "primary",
				constants.LabelMySQLClusterRole: constants.MySQLClusterRolePrimary,
			},
			matches: 1,
		}, {
			name: "blank",
			labels: map[string]string{
				constants.MySQLClusterLabel:     "blank",
				constants.LabelMySQLClusterRole: "",
			},
			matches: 1,
		}, {
			name: "missing",
			labels: map[string]string{
				constants.MySQLClusterLabel: "missing",
			},
			matches: 0,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			client := fake.NewSimpleClientset(newPodWithLabels(tt.name, tt.labels))
			pods, err := client.CoreV1().Pods(metav1.NamespaceDefault).List(metav1.ListOptions{
				LabelSelector: HasRoleSelector(tt.name).String(),
			})
			if err != nil {
				t.Fatalf("Expected no error listing pods, got: %+v", err)
			}
			if len(pods.Items) != tt.matches {
				t.Errorf("Expected %d matches for Pod with labels %+v but got %d", tt.matches, tt.labels, len(pods.Items))
			}
		})
	}
}
func newPodWithLabels(name string, labels map[string]string) *corev1.Pod {
	return &corev1.Pod{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Pod",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: metav1.NamespaceDefault,
			Labels:    labels,
		},
	}
}
