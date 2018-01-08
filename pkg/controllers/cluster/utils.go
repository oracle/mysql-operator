package cluster

import (
	"strings"

	apps "k8s.io/api/apps/v1beta1"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"

	api "github.com/oracle/mysql-operator/pkg/apis/mysql/v1"
	"github.com/oracle/mysql-operator/pkg/constants"
	statefulsets "github.com/oracle/mysql-operator/pkg/resources/statefulsets"
)

// SelectorForCluster creates a labels.Selector to match a given clusters
// associated resources.
func SelectorForCluster(c *api.MySQLCluster) labels.Selector {
	return labels.SelectorFromSet(labels.Set{constants.MySQLClusterLabel: c.Name})
}

// SelectorForClusterOperatorVersion creates a labels.Selector to match a given clusters
// associated resources MySQLOperatorVersionLabel.
func SelectorForClusterOperatorVersion(operatorVersion string) labels.Selector {
	return labels.SelectorFromSet(labels.Set{constants.MySQLOperatorVersionLabel: operatorVersion})
}

func requiresMySQLAgentStatefulSetUpgrade(ss *apps.StatefulSet, operatorVersion string) bool {
	if !SelectorForClusterOperatorVersion(operatorVersion).Matches(labels.Set(ss.Labels)) {
		return true
	}
	for _, container := range ss.Spec.Template.Spec.Containers {
		if container.Name == statefulsets.MySQLAgentContainerName {
			return extractAgentImageVersion(container.Image) != operatorVersion
		}
	}
	return false
}

func requiresMySQLAgentPodUpgrade(pod *v1.Pod, operatorVersion string) bool {
	if !SelectorForClusterOperatorVersion(operatorVersion).Matches(labels.Set(pod.Labels)) {
		return true
	}
	for _, container := range pod.Spec.Containers {
		if container.Name == statefulsets.MySQLAgentContainerName {
			return extractAgentImageVersion(container.Image) != operatorVersion
		}
	}
	return false
}

// canUpgradeMySQLAgent checks that pod can actually be updated (e.g. there no backups currently taking place).
// TODO: Implement.
func canUpgradeMySQLAgent(pod *v1.Pod) bool {
	return true
}

func extractAgentImageVersion(agentImage string) string {
	if strings.HasPrefix(agentImage, statefulsets.AgentImageName+":") {
		return strings.TrimPrefix(agentImage, statefulsets.AgentImageName+":")
	}
	return ""
}
