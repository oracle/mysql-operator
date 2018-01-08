# E2E Testing

End-to-end (e2e) testing is automated testing for real user scenarios.

## Build and run test

Prerequisites:
 - A running k8s cluster and kube config:
   - The environment variable KUBECONFIG should be set to point to the cluster config.
   - OR The environment variable KUBECONFIG_VAR should be set to contain the cluster config.
 - OCI upload yaml configuration:
   - The environment variable S3_UPLOAD_CREDS should be set to point to a file containing the backup creds as documented.
   - OR The environment variable S3_UPLOAD_CREDS_VAR should contain the backup creds as documented.
 - OCI SSH Key - This is required to SSH onto the cluster nodes and must be set-up when the OCI cluster machines are provisioned.
  - The environment variable CLUSTER_INSTANCE_SSH_KEY should be set to point to a valid key for the OCI instances the k8s cluster is running on.
   - OR The environment variable CLUSTER_INSTANCE_SSH_KEY_VAR should contain the key as documented.
 - OCI configuration:
   - The environment variable NODE_IPS should define the internal to external node IP mapping for each node in the cluster.
 - A build and pushed operator using `make push`

# Usage

You can set the environment variable "E2E_DEBUG=true" This means that if any tests fail it will not delete the cluster
to help to diagnose the failure.

There are multiple e2e targets in the makefile

 - `e2e-setup-<TestName>` - This setups the cluster with the operator and all it's requirments
 - `e2e-run-<Testname>` - This runs only the e2e test named. This requres that setup has been run first
 - `e2e-teardown-<Testname>` - This destroys all resources created during setup
 - `e2e-<Testname>` - This does the setup, run and teardown for the given the e2e test

## For Developing new tests

You probably don't want to setup and teardown every test cycle. So you can run
`make e2e-setup-<Testname>` at the begining.  Then run `make e2e-run-<Testname>` to run the tests for each iteration.
