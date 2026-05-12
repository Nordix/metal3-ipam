package e2e

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/cluster-api/test/framework"
)

func Logf(format string, a ...any) {
	fmt.Fprintf(GinkgoWriter, "INFO: "+format+"\n", a...)
}

func DumpSpecResourcesAndCleanup(ctx context.Context, specName string, bootstrapClusterProxy framework.ClusterProxy, _ framework.ClusterProxy, artifactFolder string, namespace string, intervalsGetter func(spec, key string) []any, clusterName, clusterctlLogFolder string, skipCleanup bool, clusterctlConfigPath string) {
	Expect(os.RemoveAll(clusterctlLogFolder)).To(Succeed())
	clusterClient := bootstrapClusterProxy.GetClient()

	bootstrapClusterProxy.CollectWorkloadClusterLogs(ctx, namespace, clusterName, artifactFolder)

	// Dumps all the resources in the spec namespace, then cleanups the cluster object and the spec namespace itself.
	By(fmt.Sprintf("Dumping all the Cluster API resources in the %q namespace", namespace))
	// Dump all Cluster API related resources to artifacts before deleting them.
	framework.DumpAllResources(ctx, framework.DumpAllResourcesInput{
		Lister:               clusterClient,
		Namespace:            namespace,
		LogPath:              filepath.Join(artifactFolder, bootstrapClusterProxy.GetName(), "resources"),
		KubeConfigPath:       bootstrapClusterProxy.GetKubeconfigPath(),
		ClusterctlConfigPath: clusterctlConfigPath,
	})

	if !skipCleanup {
		By(fmt.Sprintf("Deleting cluster %s/%s", namespace, clusterName))
		// While https://github.com/kubernetes-sigs/cluster-api/issues/2955 is addressed in future iterations, there is a chance
		// that cluster variable is not set even if the cluster exists, so we are calling DeleteAllClustersAndWait
		// instead of DeleteClusterAndWait
		framework.DeleteAllClustersAndWait(ctx, framework.DeleteAllClustersAndWaitInput{
			ClusterProxy:         bootstrapClusterProxy,
			Namespace:            namespace,
			ClusterctlConfigPath: clusterctlConfigPath,
			ArtifactFolder:       filepath.Join(artifactFolder, "delete-cluster"),
		}, intervalsGetter(specName, "wait-delete-cluster")...)
	}
}
