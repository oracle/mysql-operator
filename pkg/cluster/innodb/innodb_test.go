package innodb

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDeepCopyInstance(t *testing.T) {
	testCases := []struct {
		name string
		in   *Instance
	}{
		{
			name: "all-fields-populated",
			in: &Instance{
				Address: "localhost:3310",
				Mode:    "R/O",
				Role:    "HA",
				Status:  InstanceStatusOnline,
			},
		},
		{
			name: "all-fields-empty",
			in:   &Instance{},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			copy := tt.in.DeepCopy()
			assert.Equal(t, tt.in, copy)

			// Modify copy and check it is no longer equal to input (i.e. input
			// has not been mutated).
			copy.Address = "9.9.9.9"
			assert.NotEqual(t, tt.in, copy)
		})
	}
}

func TestDeepCopyReplicaSet(t *testing.T) {
	testCases := []struct {
		name string
		in   *ReplicaSet
	}{
		{
			name: "all-fields-populated",
			in: &ReplicaSet{
				Name:    "default",
				Primary: "localhost:3310",
				Status:  "OK",
				Topology: map[string]*Instance{
					"localhost:3310": &Instance{
						Address: "localhost:3310",
						Mode:    "R/O",
						Role:    "HA",
						Status:  InstanceStatusOnline,
					},
				},
			},
		},
		{
			name: "all-fields-empty",
			in:   &ReplicaSet{},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			copy := tt.in.DeepCopy()
			assert.Equal(t, tt.in, copy)

			// Modify copy and check it is no longer equal to input (i.e. input
			// has not been mutated).
			copy.Name = "non-default"
			assert.NotEqual(t, tt.in, copy)
		})
	}
}

func TestDeepCopyClusterStatus(t *testing.T) {
	testCases := []struct {
		name string
		in   *ClusterStatus
	}{
		{
			name: "all-fields-populated",
			in: &ClusterStatus{
				ClusterName: "MyCluster",
				DefaultReplicaSet: ReplicaSet{
					Name:    "MyCluster",
					Primary: "localhost:3310",
					Status:  "OK",
					Topology: map[string]*Instance{
						"localhost:3310": &Instance{
							Address: "localhost:3310",
							Mode:    "R/O",
							Role:    "HA",
							Status:  InstanceStatusOnline,
						},
					},
				},
			},
		},
		{
			name: "all-fields-empty",
			in:   &ClusterStatus{},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			copy := tt.in.DeepCopy()
			assert.Equal(t, tt.in, copy)

			// Modify copy and check it is no longer equal to input (i.e. input
			// has not been mutated).
			copy.ClusterName = "YourCluster"
			assert.NotEqual(t, tt.in, copy)
		})
	}
}
