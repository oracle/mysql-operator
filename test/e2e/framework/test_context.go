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
}

func init() {
	RegisterFlags()
}
