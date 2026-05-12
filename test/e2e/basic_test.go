package e2e

import (
	"context"
	"fmt"
	"net"

	ipamv1 "github.com/metal3-io/ip-address-manager/api/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	capipamv1 "sigs.k8s.io/cluster-api/api/ipam/v1beta2"
	"sigs.k8s.io/cluster-api/test/framework"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	ctx                 = context.TODO()
	specName            = "ipam-e2e"
	namespace           = "default"
	clusterctlLogFolder string
)

var _ = Describe("Metal3 IPAM basic functionality", Label("ipam"), func() {
	BeforeEach(func() {
		validateGlobals(specName)
		cl := bootstrapClusterProxy.GetClient()
		// Create namespace for the test if it doesn't exist
		ns := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: namespace,
			},
		}
		_ = cl.Create(ctx, ns)

		// Clean up any stale test resources from previous test runs
		cleanupTestResources(ctx, cl)
	})

	It("Should create and verify an IPPool", func() {
		By("Creating an IPPool in the bootstrap cluster")
		ipPool := createIPPool(ctx, bootstrapClusterProxy, "test-ippool-basic")

		By("Verifying that the IPPool is created successfully")
		verifyIPPool(ctx, bootstrapClusterProxy, ipPool)

		By("Cleaning up the IPPool")
		Expect(bootstrapClusterProxy.GetClient().Delete(ctx, ipPool)).To(Succeed())
	})

	It("Should allocate an IPAddress via Metal3 IPClaim", func() {
		cl := bootstrapClusterProxy.GetClient()

		By("Creating an IPPool")
		ipPool := createIPPool(ctx, bootstrapClusterProxy, "test-ippool-m3claim")

		By("Creating a Metal3 IPClaim referencing the pool")
		claimName := fmt.Sprintf("test-ipclaim-%d", GinkgoParallelProcess())
		ipClaim := createIPClaim(ctx, bootstrapClusterProxy, ipPool.Name, claimName)

		By("Waiting for the IPClaim to get an IPAddress allocated")
		Eventually(func(g Gomega) {
			retrieved := &ipamv1.IPClaim{}
			g.Expect(cl.Get(ctx, client.ObjectKeyFromObject(ipClaim), retrieved)).To(Succeed())
			g.Expect(retrieved.Status.Address).ToNot(BeNil(), "IPClaim should have an address allocated")
		}, e2eConfig.GetIntervals(specName, "wait-ippool")...).Should(Succeed())

		By("Verifying the Metal3 IPAddress object")
		updatedClaim := &ipamv1.IPClaim{}
		Expect(cl.Get(ctx, client.ObjectKeyFromObject(ipClaim), updatedClaim)).To(Succeed())

		ipAddress := &ipamv1.IPAddress{}
		Expect(cl.Get(ctx, client.ObjectKey{
			Namespace: updatedClaim.Status.Address.Namespace,
			Name:      updatedClaim.Status.Address.Name,
		}, ipAddress)).To(Succeed())

		Expect(string(ipAddress.Spec.Address)).ToNot(BeEmpty())
		ip := net.ParseIP(string(ipAddress.Spec.Address))
		Expect(ip).ToNot(BeNil(), "Allocated address should be a valid IP")
		Expect(ipAddress.Spec.Prefix).To(Equal(24))
		Expect(ipAddress.Spec.Pool.Name).To(Equal(ipPool.Name))

		By("Cleaning up IPClaim and IPPool")
		Expect(cl.Delete(ctx, ipClaim)).To(Succeed())
		Eventually(func() bool {
			err := cl.Get(ctx, client.ObjectKeyFromObject(ipClaim), &ipamv1.IPClaim{})
			return apierrors.IsNotFound(err)
		}, e2eConfig.GetIntervals(specName, "wait-ippool")...).Should(BeTrue(), "IPClaim should be deleted")
		Expect(cl.Delete(ctx, ipPool)).To(Succeed())
	})

	It("Should allocate an IPAddress via CAPI IPAddressClaim", func() {
		cl := bootstrapClusterProxy.GetClient()

		By("Creating an IPPool")
		ipPool := createIPPool(ctx, bootstrapClusterProxy, "test-ippool-capi")

		By("Creating a CAPI IPAddressClaim referencing the Metal3 IPPool")
		claimName := fmt.Sprintf("test-capi-ipclaim-%d", GinkgoParallelProcess())
		ipAddressClaim := createCAPIIPAddressClaim(ctx, bootstrapClusterProxy, ipPool.Name, claimName)

		By("Waiting for the CAPI IPAddressClaim to get an address")
		Eventually(func(g Gomega) {
			retrieved := &capipamv1.IPAddressClaim{}
			g.Expect(cl.Get(ctx, client.ObjectKeyFromObject(ipAddressClaim), retrieved)).To(Succeed())
			g.Expect(retrieved.Status.AddressRef.Name).ToNot(BeEmpty(), "IPAddressClaim should have addressRef set")
		}, e2eConfig.GetIntervals(specName, "wait-ippool")...).Should(Succeed())

		By("Verifying the CAPI IPAddress object")
		updatedClaim := &capipamv1.IPAddressClaim{}
		Expect(cl.Get(ctx, client.ObjectKeyFromObject(ipAddressClaim), updatedClaim)).To(Succeed())

		capiIPAddress := &capipamv1.IPAddress{}
		Expect(cl.Get(ctx, client.ObjectKey{
			Namespace: ipAddressClaim.Namespace,
			Name:      updatedClaim.Status.AddressRef.Name,
		}, capiIPAddress)).To(Succeed())

		Expect(capiIPAddress.Spec.Address).ToNot(BeEmpty())
		ip := net.ParseIP(capiIPAddress.Spec.Address)
		Expect(ip).ToNot(BeNil(), "Allocated address should be a valid IP")
		Expect(capiIPAddress.Spec.PoolRef.Name).To(Equal(ipPool.Name))
		Expect(capiIPAddress.Spec.PoolRef.Kind).To(Equal("IPPool"))
		Expect(capiIPAddress.Spec.PoolRef.APIGroup).To(Equal("ipam.metal3.io"))
		Expect(capiIPAddress.Spec.ClaimRef.Name).To(Equal(ipAddressClaim.Name))

		By("Cleaning up CAPI IPAddressClaim and IPPool")
		Expect(cl.Delete(ctx, ipAddressClaim)).To(Succeed())
		Eventually(func() bool {
			err := cl.Get(ctx, client.ObjectKeyFromObject(ipAddressClaim), &capipamv1.IPAddressClaim{})
			return apierrors.IsNotFound(err)
		}, e2eConfig.GetIntervals(specName, "wait-ippool")...).Should(BeTrue(), "IPAddressClaim should be deleted")
		Expect(cl.Delete(ctx, ipPool)).To(Succeed())
	})

	It("Should clean up IPAddress when Metal3 IPClaim is deleted", func() {
		cl := bootstrapClusterProxy.GetClient()

		By("Creating an IPPool and IPClaim")
		ipPool := createIPPool(ctx, bootstrapClusterProxy, "test-ippool-cleanup")
		claimName := fmt.Sprintf("test-ipclaim-cleanup-%d", GinkgoParallelProcess())
		ipClaim := createIPClaim(ctx, bootstrapClusterProxy, ipPool.Name, claimName)

		By("Waiting for IPAddress to be allocated")
		var ipAddressName string
		Eventually(func(g Gomega) {
			retrieved := &ipamv1.IPClaim{}
			g.Expect(cl.Get(ctx, client.ObjectKeyFromObject(ipClaim), retrieved)).To(Succeed())
			g.Expect(retrieved.Status.Address).ToNot(BeNil())
			ipAddressName = retrieved.Status.Address.Name
		}, e2eConfig.GetIntervals(specName, "wait-ippool")...).Should(Succeed())

		By("Deleting the IPClaim")
		Expect(cl.Delete(ctx, ipClaim)).To(Succeed())

		By("Verifying the IPClaim is removed")
		Eventually(func() bool {
			err := cl.Get(ctx, client.ObjectKeyFromObject(ipClaim), &ipamv1.IPClaim{})
			return apierrors.IsNotFound(err)
		}, e2eConfig.GetIntervals(specName, "wait-ippool")...).Should(BeTrue(), "IPClaim should be deleted")

		By("Verifying the Metal3 IPAddress is garbage collected")
		Eventually(func() bool {
			err := cl.Get(ctx, client.ObjectKey{Namespace: namespace, Name: ipAddressName}, &ipamv1.IPAddress{})
			return apierrors.IsNotFound(err)
		}, e2eConfig.GetIntervals(specName, "wait-ippool")...).Should(BeTrue(), "IPAddress should be cleaned up after IPClaim deletion")

		By("Cleaning up IPPool")
		Expect(cl.Delete(ctx, ipPool)).To(Succeed())
	})

	It("Should clean up CAPI IPAddress when IPAddressClaim is deleted", func() {
		cl := bootstrapClusterProxy.GetClient()

		By("Creating an IPPool and CAPI IPAddressClaim")
		ipPool := createIPPool(ctx, bootstrapClusterProxy, "test-ippool-capi-cleanup")
		claimName := fmt.Sprintf("test-capi-ipclaim-cleanup-%d", GinkgoParallelProcess())
		ipAddressClaim := createCAPIIPAddressClaim(ctx, bootstrapClusterProxy, ipPool.Name, claimName)

		By("Waiting for CAPI IPAddress to be allocated")
		var capiIPAddressName string
		Eventually(func(g Gomega) {
			retrieved := &capipamv1.IPAddressClaim{}
			g.Expect(cl.Get(ctx, client.ObjectKeyFromObject(ipAddressClaim), retrieved)).To(Succeed())
			g.Expect(retrieved.Status.AddressRef.Name).ToNot(BeEmpty())
			capiIPAddressName = retrieved.Status.AddressRef.Name
		}, e2eConfig.GetIntervals(specName, "wait-ippool")...).Should(Succeed())

		By("Deleting the CAPI IPAddressClaim")
		Expect(cl.Delete(ctx, ipAddressClaim)).To(Succeed())

		By("Verifying the CAPI IPAddressClaim is removed")
		Eventually(func() bool {
			err := cl.Get(ctx, client.ObjectKeyFromObject(ipAddressClaim), &capipamv1.IPAddressClaim{})
			return apierrors.IsNotFound(err)
		}, e2eConfig.GetIntervals(specName, "wait-ippool")...).Should(BeTrue(), "IPAddressClaim should be deleted")

		By("Verifying the CAPI IPAddress is garbage collected")
		Eventually(func() bool {
			err := cl.Get(ctx, client.ObjectKey{Namespace: namespace, Name: capiIPAddressName}, &capipamv1.IPAddress{})
			return apierrors.IsNotFound(err)
		}, e2eConfig.GetIntervals(specName, "wait-ippool")...).Should(BeTrue(), "CAPI IPAddress should be cleaned up after IPAddressClaim deletion")

		By("Cleaning up IPPool")
		Expect(cl.Delete(ctx, ipPool)).To(Succeed())
	})
})

// createIPPool creates an IPPool resource in the bootstrap cluster.
func createIPPool(ctx context.Context, clusterProxy framework.ClusterProxy, name string) *ipamv1.IPPool {
	Logf("Creating IPPool %s in namespace %s", name, namespace)

	startAddr := ipamv1.IPAddressStr("192.168.1.10")
	endAddr := ipamv1.IPAddressStr("192.168.1.100")
	subnet := ipamv1.IPSubnetStr("192.168.1.0/24")
	gateway := ipamv1.IPAddressStr("192.168.1.1")

	ipPool := &ipamv1.IPPool{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: ipamv1.IPPoolSpec{
			Pools: []ipamv1.Pool{
				{
					Start:   &startAddr,
					End:     &endAddr,
					Subnet:  &subnet,
					Prefix:  24,
					Gateway: &gateway,
					DNSServers: []ipamv1.IPAddressStr{
						"8.8.8.8",
						"8.8.4.4",
					},
				},
			},
			Prefix:  24,
			Gateway: &gateway,
			DNSServers: []ipamv1.IPAddressStr{
				"8.8.8.8",
				"8.8.4.4",
			},
			NamePrefix: "test-ip",
		},
	}

	Expect(clusterProxy.GetClient().Create(ctx, ipPool)).To(Succeed())
	Logf("Successfully created IPPool %s/%s", ipPool.Namespace, ipPool.Name)

	return ipPool
}

// verifyIPPool verifies that the IPPool resource exists and is properly configured.
func verifyIPPool(ctx context.Context, clusterProxy framework.ClusterProxy, ipPool *ipamv1.IPPool) {
	Logf("Verifying IPPool %s/%s", ipPool.Namespace, ipPool.Name)

	retrievedIPPool := &ipamv1.IPPool{}
	key := client.ObjectKey{
		Namespace: ipPool.Namespace,
		Name:      ipPool.Name,
	}

	Eventually(func() error {
		return clusterProxy.GetClient().Get(ctx, key, retrievedIPPool)
	}, e2eConfig.GetIntervals(specName, "wait-ippool")...).Should(Succeed(), "Failed to get IPPool")

	Expect(retrievedIPPool.Spec.ClusterName).To(BeNil())
	Expect(retrievedIPPool.Spec.Pools).To(HaveLen(1))
	Expect(string(*retrievedIPPool.Spec.Pools[0].Start)).To(Equal("192.168.1.10"))
	Expect(string(*retrievedIPPool.Spec.Pools[0].End)).To(Equal("192.168.1.100"))
	Expect(string(*retrievedIPPool.Spec.Pools[0].Subnet)).To(Equal("192.168.1.0/24"))
	Expect(retrievedIPPool.Spec.Pools[0].Prefix).To(Equal(24))
	Expect(retrievedIPPool.Spec.NamePrefix).To(Equal("test-ip"))

	Logf("Successfully verified IPPool %s/%s", ipPool.Namespace, ipPool.Name)
}

// createIPClaim creates a Metal3 IPClaim referencing the given pool name.
func createIPClaim(ctx context.Context, clusterProxy framework.ClusterProxy, poolName string, claimName string) *ipamv1.IPClaim {
	Logf("Creating IPClaim in namespace %s for pool %s", namespace, poolName)

	ipClaim := &ipamv1.IPClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      claimName,
			Namespace: namespace,
		},
		Spec: ipamv1.IPClaimSpec{
			Pool: corev1.ObjectReference{
				Name:      poolName,
				Namespace: namespace,
			},
		},
	}

	Expect(clusterProxy.GetClient().Create(ctx, ipClaim)).To(Succeed())
	Logf("Successfully created IPClaim %s/%s", ipClaim.Namespace, ipClaim.Name)

	return ipClaim
}

// createCAPIIPAddressClaim creates a CAPI IPAddressClaim (ipam.cluster.x-k8s.io/v1beta2)
// referencing a Metal3 IPPool.
func createCAPIIPAddressClaim(ctx context.Context, clusterProxy framework.ClusterProxy, poolName string, claimName string) *capipamv1.IPAddressClaim {
	Logf("Creating CAPI IPAddressClaim in namespace %s for pool %s", namespace, poolName)

	claim := &capipamv1.IPAddressClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      claimName,
			Namespace: namespace,
		},
		Spec: capipamv1.IPAddressClaimSpec{
			PoolRef: capipamv1.IPPoolReference{
				Name:     poolName,
				Kind:     "IPPool",
				APIGroup: "ipam.metal3.io",
			},
		},
	}

	Expect(clusterProxy.GetClient().Create(ctx, claim)).To(Succeed())
	Logf("Successfully created CAPI IPAddressClaim %s/%s", claim.Namespace, claim.Name)

	return claim
}

// cleanupTestResources deletes any stale test resources from previous test runs
// to prevent conflicts between parallel test executions.
func cleanupTestResources(ctx context.Context, cl client.Client) {
	// Clean up stale IPClaims (Metal3)
	ipClaimList := &ipamv1.IPClaimList{}
	if err := cl.List(ctx, ipClaimList, client.InNamespace(namespace)); err == nil {
		for i := range ipClaimList.Items {
			claim := &ipClaimList.Items[i]
			if err := cl.Delete(ctx, claim); err != nil && !apierrors.IsNotFound(err) {
				Logf("Warning: failed to delete IPClaim %s/%s: %v", claim.Namespace, claim.Name, err)
			}
		}
	}

	// Clean up stale CAPI IPAddressClaims
	capiClaimList := &capipamv1.IPAddressClaimList{}
	if err := cl.List(ctx, capiClaimList, client.InNamespace(namespace)); err == nil {
		for i := range capiClaimList.Items {
			claim := &capiClaimList.Items[i]
			if err := cl.Delete(ctx, claim); err != nil && !apierrors.IsNotFound(err) {
				Logf("Warning: failed to delete CAPI IPAddressClaim %s/%s: %v", claim.Namespace, claim.Name, err)
			}
		}
	}

	// Clean up stale IPPools
	poolList := &ipamv1.IPPoolList{}
	if err := cl.List(ctx, poolList, client.InNamespace(namespace)); err == nil {
		for i := range poolList.Items {
			pool := &poolList.Items[i]
			if err := cl.Delete(ctx, pool); err != nil && !apierrors.IsNotFound(err) {
				Logf("Warning: failed to delete IPPool %s/%s: %v", pool.Namespace, pool.Name, err)
			}
		}
	}

	// Give cleanup a moment to complete
	Eventually(func() int {
		poolCount := 0
		claimCount := 0
		addrClaimCount := 0
		if err := cl.List(ctx, poolList, client.InNamespace(namespace)); err == nil {
			poolCount = len(poolList.Items)
		}
		if err := cl.List(ctx, ipClaimList, client.InNamespace(namespace)); err == nil {
			claimCount = len(ipClaimList.Items)
		}
		if err := cl.List(ctx, capiClaimList, client.InNamespace(namespace)); err == nil {
			addrClaimCount = len(capiClaimList.Items)
		}
		return poolCount + claimCount + addrClaimCount
	}, "5s", "500ms").Should(Equal(0), "Test resources should be cleaned up")
}
