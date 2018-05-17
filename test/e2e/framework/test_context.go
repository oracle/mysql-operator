package framework

import "flag"

// TestContextType represents the co
type TestContextType struct {
	// OperatorVersion is the version of the MySQL operator under test.
	OperatorVersion string

	// RepoRoot is the root directory of the repository.
	RepoRoot string

	// KubeConfig is the path to the kubeconfig file.
	KubeConfig string

	// Namespace (if provided) is the namespace of an existing namespace to
	// use for test execution rather than creating a new namespace.
	Namespace string
	// DeleteNamespace controls whether or not to delete test namespaces
	DeleteNamespace bool
	// DeleteNamespaceOnFailure controls whether or not to delete test
	// namespaces when the test fails.
	DeleteNamespaceOnFailure bool

	// S3AccessKey is the S3 (compat.) access key for the bucket used in
	// backup / restore tests.
	S3AccessKey string
	// S3AccessKey is the S3 (compat.) secret key for the bucket used in
	// backup / restore tests.
	S3SecretKey string
}

// TestContext holds the context of the the test run.
var TestContext TestContextType

// RegisterFlags registers the test framework flags and populates TestContext.
func RegisterFlags() {
	flag.StringVar(&TestContext.RepoRoot, "repo-root", "../../", "Root directory of kubernetes repository, for finding test files.")
	flag.StringVar(&TestContext.OperatorVersion, "operator-version", "", "The version of the MySQL operator under test")
	flag.StringVar(&TestContext.KubeConfig, "kubeconfig", "", "Path to Kubeconfig file with authorization and master location information.")
	flag.StringVar(&TestContext.Namespace, "namespace", "", "Name of an existing Namespace to run tests in")
	flag.BoolVar(&TestContext.DeleteNamespace, "delete-namespace", true, "If true tests will delete namespace after completion. It is only designed to make debugging easier, DO NOT turn it off by default.")
	flag.BoolVar(&TestContext.DeleteNamespaceOnFailure, "delete-namespace-on-failure", true, "If true tests will delete their associated namespace upon completion whether or not the test has failed.")
	flag.StringVar(&TestContext.S3AccessKey, "s3-access-key", "", "The S3 (compat.) access key for the bucket used in backup / restore tests.")
	flag.StringVar(&TestContext.S3SecretKey, "s3-secret-key", "", "The S3 (compat.) secret key for the bucket used in backup / restore tests.")
}

func init() {
	RegisterFlags()
}
