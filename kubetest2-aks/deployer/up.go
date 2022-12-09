/*
Copyright 2022 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package deployer

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/containerservice/armcontainerservice/v2"
	armcontainerservicev2 "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/containerservice/armcontainerservice/v2"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources"
	"github.com/Azure/go-autorest/autorest/to"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog"

	"sigs.k8s.io/kubetest2/pkg/exec"
)

var (
	apiVersion           = "2022-03-01"
	defaultKubeconfigDir = "_kubeconfig"
	usageTag             = "aks-cluster-e2e"
	cred                 *azidentity.DefaultAzureCredential
	onceWrapper          sync.Once
)

type UpOptions struct {
	ClusterName      string `flag:"clusterName" desc:"--clusterName flag for aks cluster name"`
	Location         string `flag:"location" desc:"--location flag for resource group and cluster location"`
	CCMImageTag      string `flag:"ccmImageTag" desc:"--ccmImageTag flag for CCM image tag"`
	ConfigPath       string `flag:"config" desc:"--config flag for AKS cluster"`
	CustomConfigPath string `flag:"customConfig" desc:"--customConfig flag for custom configuration"`
	K8sVersion       string `flag:"k8sVersion" desc:"--k8sVersion flag for cluster Kubernetes version"`
}

type enableCustomFeaturesPolicy struct{}

type changeLocationPolicy struct{}

type addCustomConfigPolicy struct {
	customConfig string
}

// A copy of armcontainerservice.ManagedClusterProperties
type Kubetest2ManagedClusterProperties struct {
	// The Azure Active Directory configuration.
	AADProfile *armcontainerservice.ManagedClusterAADProfile `json:"aadProfile,omitempty"`

	// The access profile for managed cluster API server.
	APIServerAccessProfile *armcontainerservice.ManagedClusterAPIServerAccessProfile `json:"apiServerAccessProfile,omitempty"`

	// The profile of managed cluster add-on.
	AddonProfiles map[string]*armcontainerservice.ManagedClusterAddonProfile `json:"addonProfiles,omitempty"`

	// The agent pool properties.
	AgentPoolProfiles []*armcontainerservice.ManagedClusterAgentPoolProfile `json:"agentPoolProfiles,omitempty"`

	// Parameters to be applied to the cluster-autoscaler when enabled
	AutoScalerProfile *armcontainerservice.ManagedClusterPropertiesAutoScalerProfile `json:"autoScalerProfile,omitempty"`

	// The auto upgrade configuration.
	AutoUpgradeProfile *armcontainerservice.ManagedClusterAutoUpgradeProfile `json:"autoUpgradeProfile,omitempty"`

	// This cannot be updated once the Managed Cluster has been created.
	DNSPrefix *string `json:"dnsPrefix,omitempty"`

	// If set to true, getting static credentials will be disabled for this cluster. This must only be used on Managed Clusters
	// that are AAD enabled. For more details see disable local accounts
	// [https://docs.microsoft.com/azure/aks/managed-aad#disable-local-accounts-preview].
	DisableLocalAccounts *bool `json:"disableLocalAccounts,omitempty"`

	// This is of the form: '/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Compute/diskEncryptionSets/{encryptionSetName}'
	DiskEncryptionSetID *string `json:"diskEncryptionSetID,omitempty"`

	// (DEPRECATING) Whether to enable Kubernetes pod security policy (preview). This feature is set for removal on October 15th,
	// 2020. Learn more at aka.ms/aks/azpodpolicy.
	EnablePodSecurityPolicy *bool `json:"enablePodSecurityPolicy,omitempty"`

	// Whether to enable Kubernetes Role-Based Access Control.
	EnableRBAC *bool `json:"enableRBAC,omitempty"`

	// This cannot be updated once the Managed Cluster has been created.
	FqdnSubdomain *string `json:"fqdnSubdomain,omitempty"`

	// Configurations for provisioning the cluster with HTTP proxy servers.
	HTTPProxyConfig *armcontainerservice.ManagedClusterHTTPProxyConfig `json:"httpProxyConfig,omitempty"`

	// Identities associated with the cluster.
	IdentityProfile map[string]*armcontainerservice.UserAssignedIdentity `json:"identityProfile,omitempty"`

	// Both patch version (e.g. 1.20.13) and (e.g. 1.20) are supported. When is specified, the latest supported GA patch version
	// is chosen automatically. Updating the cluster with the same once it has been
	// created (e.g. 1.14.x -> 1.14) will not trigger an upgrade, even if a newer patch version is available. When you upgrade
	// a supported AKS cluster, Kubernetes minor versions cannot be skipped. All
	// upgrades must be performed sequentially by major version number. For example, upgrades between 1.14.x -> 1.15.x or 1.15.x
	// -> 1.16.x are allowed, however 1.14.x -> 1.16.x is not allowed. See upgrading
	// an AKS cluster [https://docs.microsoft.com/azure/aks/upgrade-cluster] for more details.
	KubernetesVersion *string `json:"kubernetesVersion,omitempty"`

	// The profile for Linux VMs in the Managed Cluster.
	LinuxProfile *armcontainerservice.LinuxProfile `json:"linuxProfile,omitempty"`

	// The network configuration profile.
	NetworkProfile *armcontainerservice.NetworkProfile `json:"networkProfile,omitempty"`

	// The name of the resource group containing agent pool nodes.
	NodeResourceGroup *string `json:"nodeResourceGroup,omitempty"`

	// The OIDC issuer profile of the Managed Cluster.
	OidcIssuerProfile *armcontainerservice.ManagedClusterOIDCIssuerProfile `json:"oidcIssuerProfile,omitempty"`

	// See use AAD pod identity [https://docs.microsoft.com/azure/aks/use-azure-ad-pod-identity] for more details on AAD pod identity
	// integration.
	PodIdentityProfile *armcontainerservice.ManagedClusterPodIdentityProfile `json:"podIdentityProfile,omitempty"`

	// Private link resources associated with the cluster.
	PrivateLinkResources []*armcontainerservice.PrivateLinkResource `json:"privateLinkResources,omitempty"`

	// Allow or deny public network access for AKS
	PublicNetworkAccess *armcontainerservice.PublicNetworkAccess `json:"publicNetworkAccess,omitempty"`

	// Security profile for the managed cluster.
	SecurityProfile *armcontainerservice.ManagedClusterSecurityProfile `json:"securityProfile,omitempty"`

	// Information about a service principal identity for the cluster to use for manipulating Azure APIs.
	ServicePrincipalProfile *armcontainerservice.ManagedClusterServicePrincipalProfile `json:"servicePrincipalProfile,omitempty"`

	// Storage profile for the managed cluster.
	StorageProfile *armcontainerservice.ManagedClusterStorageProfile `json:"storageProfile,omitempty"`

	// The profile for Windows VMs in the Managed Cluster.
	WindowsProfile *armcontainerservice.ManagedClusterWindowsProfile `json:"windowsProfile,omitempty"`

	// READ-ONLY; The Azure Portal requires certain Cross-Origin Resource Sharing (CORS) headers to be sent in some responses,
	// which Kubernetes APIServer doesn't handle by default. This special FQDN supports CORS,
	// allowing the Azure Portal to function properly.
	AzurePortalFQDN *string `json:"azurePortalFQDN,omitempty" azure:"ro"`

	// READ-ONLY; If kubernetesVersion was a fully specified version , this field will be exactly equal to it. If kubernetesVersion
	// was , this field will contain the full version being used.
	CurrentKubernetesVersion *string `json:"currentKubernetesVersion,omitempty" azure:"ro"`

	// READ-ONLY; The FQDN of the master pool.
	Fqdn *string `json:"fqdn,omitempty" azure:"ro"`

	// READ-ONLY; The max number of agent pools for the managed cluster.
	MaxAgentPools *int32 `json:"maxAgentPools,omitempty" azure:"ro"`

	// READ-ONLY; The Power State of the cluster.
	PowerState *armcontainerservice.PowerState `json:"powerState,omitempty" azure:"ro"`

	// READ-ONLY; The FQDN of private cluster.
	PrivateFQDN *string `json:"privateFQDN,omitempty" azure:"ro"`

	// READ-ONLY; The current provisioning state.
	ProvisioningState *string `json:"provisioningState,omitempty" azure:"ro"`

	EncodedCustomConfiguration *string `json:"encodedCustomConfiguration,omitempty"`
}

// A copy of armcontainerservice.MangedCluster
type kubetest2ManagedCluster struct {
	Location *string `json:"location,omitempty"`

	// The extended location of the Virtual Machine.
	ExtendedLocation *armcontainerservice.ExtendedLocation `json:"extendedLocation,omitempty"`

	// The identity of the managed cluster, if configured.
	Identity *armcontainerservice.ManagedClusterIdentity `json:"identity,omitempty"`

	// Properties of a managed cluster.
	Properties *Kubetest2ManagedClusterProperties `json:"properties,omitempty"`

	// The managed cluster SKU.
	SKU *armcontainerservice.ManagedClusterSKU `json:"sku,omitempty"`

	// Resource tags.
	Tags map[string]*string `json:"tags,omitempty"`

	// READ-ONLY; Fully qualified resource ID for the resource. Ex - /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/{resourceProviderNamespace}/{resourceType}/{resourceName}
	ID *string `json:"id,omitempty" azure:"ro"`

	// READ-ONLY; The name of the resource
	Name *string `json:"name,omitempty" azure:"ro"`

	// READ-ONLY; Azure Resource Manager metadata containing createdBy and modifiedBy information.
	SystemData *armcontainerservice.SystemData `json:"systemData,omitempty" azure:"ro"`

	// READ-ONLY; The type of the resource. E.g. "Microsoft.Compute/virtualMachines" or "Microsoft.Storage/storageAccounts"
	Type *string `json:"type,omitempty" azure:"ro"`
}

func (p enableCustomFeaturesPolicy) Do(req *policy.Request) (*http.Response, error) {
	req.Raw().Header.Add("AKSHTTPCustomFeatures", "Microsoft.ContainerService/EnableCloudControllerManager")
	return req.Next()
}

func (p changeLocationPolicy) Do(req *policy.Request) (*http.Response, error) {
	if strings.Contains(req.Raw().URL.String(), "Microsoft.ContainerService/locations") {
		q := req.Raw().URL.Query()
		q.Set("api-version", "2017-08-31")
		req.Raw().URL.RawQuery = q.Encode()
	}

	return req.Next()
}

func (p addCustomConfigPolicy) Do(req *policy.Request) (*http.Response, error) {
	body, err := ioutil.ReadAll(req.Raw().Body)
	defer req.Raw().Body.Close()
	if err != nil {
		return nil, err
	}

	managedCluster := kubetest2ManagedCluster{}
	if err = json.Unmarshal([]byte(body), &managedCluster); err != nil {
		return nil, fmt.Errorf("failed to unmarshal managed cluster config: %v", err)
	}
	managedCluster.Properties.EncodedCustomConfiguration = &p.customConfig

	var encodedMangedCluster []byte
	if encodedMangedCluster, err = json.Marshal(managedCluster); err != nil {
		return nil, fmt.Errorf("failed to marshal managed cluster config: %v", err)
	}

	if _, err := req.Raw().Body.Read(encodedMangedCluster); err != nil {
		return nil, fmt.Errorf("failed to read body of encodedMangedCluster: %v, body:\n%s", err, encodedMangedCluster)
	}

	// req.SetBody(exported.NopCloser(bytes.NewReader(encodedMangedCluster)), shared.ContentTypeAppJSON)

	// _, _ = req.Raw().Body.Read(encodedMangedCluster)
	// _, _ = req.Raw().Body.Read([]byte{})

	return req.Next()
}

func runCmd(cmd exec.Cmd) error {
	exec.InheritOutput(cmd)
	return cmd.Run()
}

func init() {
	onceWrapper.Do(func() {
		var err error
		cred, err = azidentity.NewDefaultAzureCredential(nil)
		if err != nil {
			klog.Fatalf("failed to authenticate: %v", err)
		}
	})
}

// Define the function to create a resource group.
func (d *deployer) createResourceGroup(subscriptionID string) (armresources.ResourceGroupsClientCreateOrUpdateResponse, error) {
	rgClient, _ := armresources.NewResourceGroupsClient(subscriptionID, cred, nil)

	now := time.Now()
	timestamp := now.Unix()
	param := armresources.ResourceGroup{
		Location: to.StringPtr(d.Location),
		Tags: map[string]*string{
			"creation_date": to.StringPtr(fmt.Sprintf("%d", timestamp)),
			"usage":         to.StringPtr(usageTag),
		},
	}

	return rgClient.CreateOrUpdate(ctx, d.ResourceGroupName, param, nil)
}

func openPath(path string) ([]byte, error) {
	if !strings.HasPrefix(path, "http://") && !strings.HasPrefix(path, "https://") {
		return ioutil.ReadFile(path)
	}
	resp, err := http.Get(path)
	if err != nil {
		return []byte{}, fmt.Errorf("failed to http get url: %s", path)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return []byte{}, fmt.Errorf("failed to http get url with StatusCode: %d", resp.StatusCode)
	}

	return ioutil.ReadAll(resp.Body)
}

// prepareClusterConfig generates cluster config.
func (d *deployer) prepareClusterConfig(imageTag string, clusterID string) (string, string, error) {
	configFile, err := openPath(d.ConfigPath)
	if err != nil {
		return "", "", fmt.Errorf("failed to read cluster config file at %q: %v", d.ConfigPath, err)
	}
	clusterConfig := string(configFile)
	clusterConfigMap := map[string]string{
		"{AKS_CLUSTER_ID}":      clusterID,
		"{CLUSTER_NAME}":        d.ClusterName,
		"{AZURE_LOCATION}":      d.Location,
		"{AZURE_CLIENT_ID}":     clientID,
		"{AZURE_CLIENT_SECRET}": clientSecret,
		"{KUBERNETES_VERSION}":  d.K8sVersion,
	}
	for k, v := range clusterConfigMap {
		clusterConfig = strings.ReplaceAll(clusterConfig, k, v)
	}

	customConfig, err := openPath(d.CustomConfigPath)
	if err != nil {
		return "", "", fmt.Errorf("failed to read custom config file at %q: %v", d.CustomConfigPath, err)
	}

	cloudProviderImageMap := map[string]string{
		"{CUSTOM_CCM_IMAGE}": fmt.Sprintf("%s/azure-cloud-controller-manager:%s", imageRegistry, imageTag),
		"{CUSTOM_CNM_IMAGE}": fmt.Sprintf("%s/azure-cloud-node-manager:%s-linux-amd64", imageRegistry, imageTag),
	}
	for k, v := range cloudProviderImageMap {
		customConfig = bytes.ReplaceAll(customConfig, []byte(k), []byte(v))
	}
	klog.Infof("AKS cluster custom config: %s", customConfig)

	encodedCustomConfig := base64.StdEncoding.EncodeToString(customConfig)
	clusterConfig = strings.ReplaceAll(clusterConfig, "{CUSTOM_CONFIG}", encodedCustomConfig)

	return clusterConfig, encodedCustomConfig, nil
}

type FuncPolicyWrapper func(req *policy.Request) (*http.Response, error)

func (f FuncPolicyWrapper) Do(req *policy.Request) (*http.Response, error) {
	if f != nil {
		return f(req)
	}
	return nil, nil
}

// createAKSWithCustomConfig creates an AKS cluster with custom configuration.
func (d *deployer) createAKSWithCustomConfig(imageTag string) error {
	klog.Infof("Creating the AKS cluster with custom config")
	clusterID := fmt.Sprintf("/subscriptions/%s/resourcegroups/%s/providers/Microsoft.ContainerService/managedClusters/%s", subscriptionID, d.ResourceGroupName, d.ClusterName)

	clusterConfig, encodedCustomConfig, err := d.prepareClusterConfig(imageTag, clusterID)
	if err != nil {
		return fmt.Errorf("failed to prepare cluster config: %v", err)
	}
	klog.Infof("AKS cluster config: %s", clusterConfig)
	options := arm.ClientOptions{
		ClientOptions: policy.ClientOptions{
			APIVersion: apiVersion,
			PerCallPolicies: []policy.Policy{
				&enableCustomFeaturesPolicy{},
				&changeLocationPolicy{},
				&addCustomConfigPolicy{encodedCustomConfig},
				// FuncPolicyWrapper(func(req *policy.Request) (*http.Response, error) {
				// 	if req.Raw().Method == http.MethodPut {
				// 		req.Raw().Body
				// 	}
				// }),
			},
		},
		DisableRPRegistration: false,
	}

	client, err := armcontainerservicev2.NewManagedClustersClient(subscriptionID, cred, &options)
	if err != nil {
		return fmt.Errorf("failed to new managed cluster client with sub ID %q: %v", subscriptionID, err)
	}

	mcConfig := armcontainerservicev2.ManagedCluster{}
	if err = json.Unmarshal([]byte(clusterConfig), &mcConfig); err != nil {
		return fmt.Errorf("failed to unmarshal cluster config: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()

	poller, err := client.BeginCreateOrUpdate(ctx, d.ResourceGroupName, d.ClusterName, mcConfig, nil)
	if err != nil {
		return fmt.Errorf("failed to put resource: %v", err.Error())
	}
	_, err = poller.PollUntilDone(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to put resource: %v", err.Error())
	}

	klog.Infof("An AKS cluster %q in resource group %q is creating", d.ClusterName, d.ResourceGroupName)
	return nil
}

// getAKSKubeconfig gets kubeconfig of the AKS cluster and writes it to specific path.
func (d *deployer) getAKSKubeconfig() error {
	klog.Infof("Retrieving AKS cluster's kubeconfig")
	client, err := armcontainerservicev2.NewManagedClustersClient(subscriptionID, cred, nil)
	if err != nil {
		return fmt.Errorf("failed to new managed cluster client with sub ID %q: %v", subscriptionID, err)
	}

	var resp armcontainerservicev2.ManagedClustersClientListClusterUserCredentialsResponse
	err = wait.PollImmediate(1*time.Minute, 20*time.Minute, func() (done bool, err error) {
		resp, err = client.ListClusterUserCredentials(ctx, d.ResourceGroupName, d.ClusterName, nil)
		if err != nil {
			if strings.Contains(err.Error(), "404 Not Found") {
				klog.Infof("failed to list cluster user credentials for 1 minute, retrying")
				return false, nil
			}
			return false, fmt.Errorf("failed to list cluster user credentials with resource group name %q, cluster ID %q: %v", d.ResourceGroupName, d.ClusterName, err)
		}
		return true, nil
	})
	if err != nil {
		return err
	}

	kubeconfigs := resp.CredentialResults.Kubeconfigs
	if len(kubeconfigs) == 0 {
		return fmt.Errorf("failed to find a valid kubeconfig")
	}
	kubeconfig := kubeconfigs[0]
	destPath := fmt.Sprintf("%s/%s_%s.kubeconfig", defaultKubeconfigDir, d.ResourceGroupName, d.ClusterName)

	if err := os.MkdirAll(defaultKubeconfigDir, os.ModePerm); err != nil {
		return fmt.Errorf("failed to mkdir the default kubeconfig dir: %v", err)
	}
	if err := ioutil.WriteFile(destPath, kubeconfig.Value, 0666); err != nil {
		return fmt.Errorf("failed to write kubeconfig to %s", destPath)
	}

	klog.Infof("Succeeded in getting kubeconfig of cluster %q in resource group %q", d.ClusterName, d.ResourceGroupName)
	return nil
}

func (d *deployer) verifyUpFlags() error {
	if d.ResourceGroupName == "" {
		return fmt.Errorf("resource group name is empty")
	}
	if d.Location == "" {
		return fmt.Errorf("location is empty")
	}
	if d.ClusterName == "" {
		d.ClusterName = "aks-cluster"
	}
	if d.ConfigPath == "" {
		return fmt.Errorf("cluster config path is empty")
	}
	if d.CustomConfigPath == "" {
		return fmt.Errorf("custom config path is empty")
	}
	if d.CCMImageTag == "" {
		return fmt.Errorf("ccm image tag is empty")
	}
	if d.K8sVersion == "" {
		return fmt.Errorf("k8s version is empty")
	}
	return nil
}

func (d *deployer) Up() error {
	if err := d.verifyUpFlags(); err != nil {
		return fmt.Errorf("up flags are invalid: %v", err)
	}

	// Create the resource group
	resourceGroup, err := d.createResourceGroup(subscriptionID)
	if err != nil {
		return fmt.Errorf("failed to create the resource group: %v", err)
	}
	klog.Infof("Resource group %s created", *resourceGroup.ResourceGroup.ID)

	// Create the AKS cluster
	if err := d.createAKSWithCustomConfig(d.CCMImageTag); err != nil {
		return fmt.Errorf("failed to create the AKS cluster: %v", err)
	}

	// Wait for the cluster to be up
	if err := d.waitForClusterUp(); err != nil {
		return fmt.Errorf("failed to wait for cluster to be up: %v", err)
	}

	// Get the cluster kubeconfig
	if err := d.getAKSKubeconfig(); err != nil {
		return fmt.Errorf("failed to get AKS cluster kubeconfig: %v", err)
	}
	return nil
}

func (d *deployer) waitForClusterUp() error {
	klog.Infof("Waiting for AKS cluster to be up")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	client, err := armcontainerservicev2.NewManagedClustersClient(subscriptionID, cred, nil)
	if err != nil {
		return fmt.Errorf("failed to new managed cluster client with sub ID %q: %v", subscriptionID, err)
	}
	err = wait.PollImmediate(10*time.Second, 10*time.Minute, func() (done bool, err error) {
		managedCluster, rerr := client.Get(ctx, d.ResourceGroupName, d.ClusterName, nil)
		if rerr != nil {
			return false, fmt.Errorf("failed to get managed cluster %q in resource group %q: %v", d.ClusterName, d.ResourceGroupName, rerr.Error())
		}
		return managedCluster.Properties.ProvisioningState != nil && *managedCluster.Properties.ProvisioningState == "Succeeded", nil
	})
	return err
}

func (d *deployer) IsUp() (up bool, err error) {
	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		klog.Fatalf("failed to authenticate: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	client, err := armcontainerservicev2.NewManagedClustersClient(subscriptionID, cred, nil)
	if err != nil {
		return false, fmt.Errorf("failed to new managed cluster client with sub ID %q: %v", subscriptionID, err)
	}
	managedCluster, rerr := client.Get(ctx, d.ResourceGroupName, d.ClusterName, nil)
	if rerr != nil {
		return false, fmt.Errorf("failed to get managed cluster %q in resource group %q: %v", d.ClusterName, d.ResourceGroupName, rerr.Error())
	}

	return managedCluster.Properties.ProvisioningState != nil && *managedCluster.Properties.ProvisioningState == "Succeeded", nil
}
