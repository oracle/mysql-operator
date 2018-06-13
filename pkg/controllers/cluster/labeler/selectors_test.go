// Copyright 2018 Oracle and/or its affiliates. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
				constants.ClusterLabel:     "primary",
				constants.LabelClusterRole: constants.ClusterRolePrimary,
			},
			matches: 1,
		}, {
			name: "secondary",
			labels: map[string]string{
				constants.ClusterLabel:     "secondary",
				constants.LabelClusterRole: constants.ClusterRoleSecondary,
			},
			matches: 0,
		}, {
			name: "blank",
			labels: map[string]string{
				constants.ClusterLabel:     "blank",
				constants.LabelClusterRole: "",
			},
			matches: 0,
		}, {
			name: "missing",
			labels: map[string]string{
				constants.ClusterLabel: "missing",
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
				constants.ClusterLabel:     "secondary",
				constants.LabelClusterRole: constants.ClusterRoleSecondary,
			},
			matches: 1,
		}, {
			name: "primary",
			labels: map[string]string{
				constants.ClusterLabel:     "primary",
				constants.LabelClusterRole: constants.ClusterRolePrimary,
			},
			matches: 0,
		}, {
			name: "blank",
			labels: map[string]string{
				constants.ClusterLabel:     "blank",
				constants.LabelClusterRole: "",
			},
			matches: 0,
		}, {
			name: "missing",
			labels: map[string]string{
				constants.ClusterLabel: "missing",
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
				constants.ClusterLabel:     "secondary",
				constants.LabelClusterRole: constants.ClusterRoleSecondary,
			},
			matches: 1,
		}, {
			name: "primary",
			labels: map[string]string{
				constants.ClusterLabel:     "primary",
				constants.LabelClusterRole: constants.ClusterRolePrimary,
			},
			matches: 0,
		}, {
			name: "blank",
			labels: map[string]string{
				constants.ClusterLabel:     "blank",
				constants.LabelClusterRole: "",
			},
			matches: 1,
		}, {
			name: "missing",
			labels: map[string]string{
				constants.ClusterLabel: "missing",
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
				constants.ClusterLabel:     "secondary",
				constants.LabelClusterRole: constants.ClusterRoleSecondary,
			},
			matches: 1,
		}, {
			name: "primary",
			labels: map[string]string{
				constants.ClusterLabel:     "primary",
				constants.LabelClusterRole: constants.ClusterRolePrimary,
			},
			matches: 1,
		}, {
			name: "blank",
			labels: map[string]string{
				constants.ClusterLabel:     "blank",
				constants.LabelClusterRole: "",
			},
			matches: 1,
		}, {
			name: "missing",
			labels: map[string]string{
				constants.ClusterLabel: "missing",
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
