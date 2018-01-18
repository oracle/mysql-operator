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

package constants

// MySQLClusterLabel is applied to all components of a MySQL cluster
const MySQLClusterLabel = "v1.mysql.oracle.com/cluster"

// MySQLOperatorVersionLabel denotes the version of the MySQLOperator and
// MySQLOperatorAgent running in the cluster.
const MySQLOperatorVersionLabel = "v1.mysql.oracle.com/version"

// LabelMySQLClusterRole specifies the role of a Pod within a MySQLCluster.
const LabelMySQLClusterRole = "v1.mysql.oracle.com/role"

// MySQLClusterRolePrimary denotes a primary InnoDB cluster member.
const MySQLClusterRolePrimary = "primary"

// MySQLClusterRoleSecondary denotes a secondary InnoDB cluster member.
const MySQLClusterRoleSecondary = "secondary"
