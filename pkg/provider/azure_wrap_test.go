/*
Copyright 2020 The Kubernetes Authors.

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

package provider

import (
	"net/http"
	"reflect"
	"sync"
	"testing"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2022-08-01/compute"
	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2022-07-01/network"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/utils/pointer"

	"sigs.k8s.io/cloud-provider-azure/pkg/azureclients/loadbalancerclient/mockloadbalancerclient"
	"sigs.k8s.io/cloud-provider-azure/pkg/azureclients/privatelinkserviceclient/mockprivatelinkserviceclient"
	"sigs.k8s.io/cloud-provider-azure/pkg/azureclients/publicipclient/mockpublicipclient"
	"sigs.k8s.io/cloud-provider-azure/pkg/azureclients/routetableclient/mockroutetableclient"
	"sigs.k8s.io/cloud-provider-azure/pkg/azureclients/securitygroupclient/mocksecuritygroupclient"
	"sigs.k8s.io/cloud-provider-azure/pkg/azureclients/vmclient/mockvmclient"
	azcache "sigs.k8s.io/cloud-provider-azure/pkg/cache"
	"sigs.k8s.io/cloud-provider-azure/pkg/consts"
	"sigs.k8s.io/cloud-provider-azure/pkg/retry"
)

func TestExtractNotFound(t *testing.T) {
	notFound := &retry.Error{HTTPStatusCode: http.StatusNotFound}
	otherHTTP := &retry.Error{HTTPStatusCode: http.StatusForbidden}
	otherErr := &retry.Error{HTTPStatusCode: http.StatusTooManyRequests}

	tests := []struct {
		err         *retry.Error
		expectedErr *retry.Error
		exists      bool
	}{
		{nil, nil, true},
		{otherErr, otherErr, false},
		{notFound, nil, false},
		{otherHTTP, otherHTTP, false},
	}

	for _, test := range tests {
		exists, err := checkResourceExistsFromError(test.err)
		if test.exists != exists {
			t.Errorf("expected: %v, saw: %v", test.exists, exists)
		}
		if !reflect.DeepEqual(test.expectedErr, err) {
			t.Errorf("expected err: %v, saw: %v", test.expectedErr, err)
		}
	}
}

func TestIsNodeUnmanaged(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tests := []struct {
		name           string
		unmanagedNodes sets.Set[string]
		node           string
		expected       bool
		expectErr      bool
	}{
		{
			name:           "unmanaged node should return true",
			unmanagedNodes: sets.New("node1", "node2"),
			node:           "node1",
			expected:       true,
		},
		{
			name:           "managed node should return false",
			unmanagedNodes: sets.New("node1", "node2"),
			node:           "node3",
			expected:       false,
		},
		{
			name:           "empty unmanagedNodes should return true",
			unmanagedNodes: sets.New[string](),
			node:           "node3",
			expected:       false,
		},
		{
			name:           "no synced informer should report error",
			unmanagedNodes: sets.New[string](),
			node:           "node1",
			expectErr:      true,
		},
	}

	az := GetTestCloud(ctrl)
	for _, test := range tests {
		az.unmanagedNodes = test.unmanagedNodes
		if test.expectErr {
			az.nodeInformerSynced = func() bool {
				return false
			}
		}

		real, err := az.IsNodeUnmanaged(test.node)
		if test.expectErr {
			assert.Error(t, err, test.name)
			continue
		}

		assert.NoError(t, err, test.name)
		assert.Equal(t, test.expected, real, test.name)
	}
}

func TestIsNodeUnmanagedByProviderID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tests := []struct {
		providerID string
		expected   bool
		name       string
	}{
		{
			providerID: consts.CloudProviderName + ":///subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/myResourceGroupName/providers/Microsoft.Compute/virtualMachines/k8s-agent-AAAAAAAA-0",
			expected:   false,
		},
		{
			providerID: consts.CloudProviderName + "://",
			expected:   true,
		},
		{
			providerID: ":///subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/myResourceGroupName/providers/Microsoft.Compute/virtualMachines/k8s-agent-AAAAAAAA-0",
			expected:   true,
		},
		{
			providerID: "aws:///subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/myResourceGroupName/providers/Microsoft.Compute/virtualMachines/k8s-agent-AAAAAAAA-0",
			expected:   true,
		},
		{
			providerID: "k8s-agent-AAAAAAAA-0",
			expected:   true,
		},
	}

	az := GetTestCloud(ctrl)
	for _, test := range tests {
		isUnmanagedNode := az.IsNodeUnmanagedByProviderID(test.providerID)
		assert.Equal(t, test.expected, isUnmanagedNode, test.providerID)
	}
}

func TestConvertResourceGroupNameToLower(t *testing.T) {
	tests := []struct {
		desc        string
		resourceID  string
		expected    string
		expectError bool
	}{
		{
			desc:        "empty string should report error",
			resourceID:  "",
			expectError: true,
		},
		{
			desc:        "resourceID not in Azure format should report error",
			resourceID:  "invalid-id",
			expectError: true,
		},
		{
			desc:        "providerID not in Azure format should report error",
			resourceID:  "azure://invalid-id",
			expectError: true,
		},
		{
			desc:       "resource group name in VM providerID should be converted",
			resourceID: consts.CloudProviderName + ":///subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/myResourceGroupName/providers/Microsoft.Compute/virtualMachines/k8s-agent-AAAAAAAA-0",
			expected:   consts.CloudProviderName + ":///subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/myresourcegroupname/providers/Microsoft.Compute/virtualMachines/k8s-agent-AAAAAAAA-0",
		},
		{
			desc:       "resource group name in VM resourceID should be converted",
			resourceID: "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/myResourceGroupName/providers/Microsoft.Compute/virtualMachines/k8s-agent-AAAAAAAA-0",
			expected:   "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/myresourcegroupname/providers/Microsoft.Compute/virtualMachines/k8s-agent-AAAAAAAA-0",
		},
		{
			desc:       "resource group name in VMSS providerID should be converted",
			resourceID: consts.CloudProviderName + ":///subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/myResourceGroupName/providers/Microsoft.Compute/virtualMachineScaleSets/myScaleSetName/virtualMachines/156",
			expected:   consts.CloudProviderName + ":///subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/myresourcegroupname/providers/Microsoft.Compute/virtualMachineScaleSets/myScaleSetName/virtualMachines/156",
		},
		{
			desc:       "resource group name in VMSS resourceID should be converted",
			resourceID: "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/myResourceGroupName/providers/Microsoft.Compute/virtualMachineScaleSets/myScaleSetName/virtualMachines/156",
			expected:   "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/myresourcegroupname/providers/Microsoft.Compute/virtualMachineScaleSets/myScaleSetName/virtualMachines/156",
		},
	}

	for _, test := range tests {
		real, err := ConvertResourceGroupNameToLower(test.resourceID)
		if test.expectError {
			assert.NotNil(t, err, test.desc)
			continue
		}

		assert.Nil(t, err, test.desc)
		assert.Equal(t, test.expected, real, test.desc)
	}
}

func TestIsBackendPoolOnSameLB(t *testing.T) {
	tests := []struct {
		backendPoolID        string
		expectedLBName       string
		existingBackendPools []string
		expected             bool
		expectError          bool
	}{
		{
			backendPoolID: "/subscriptions/sub/resourceGroups/rg/providers/Microsoft.Network/loadBalancers/lb1/backendAddressPools/pool1",
			existingBackendPools: []string{
				"/subscriptions/sub/resourceGroups/rg/providers/Microsoft.Network/loadBalancers/lb1/backendAddressPools/pool2",
			},
			expected:       true,
			expectedLBName: "",
		},
		{
			backendPoolID: "/subscriptions/sub/resourceGroups/rg/providers/Microsoft.Network/loadBalancers/lb1-internal/backendAddressPools/pool1",
			existingBackendPools: []string{
				"/subscriptions/sub/resourceGroups/rg/providers/Microsoft.Network/loadBalancers/lb1/backendAddressPools/pool2",
			},
			expected:       true,
			expectedLBName: "",
		},
		{
			backendPoolID: "/subscriptions/sub/resourceGroups/rg/providers/Microsoft.Network/loadBalancers/lb1/backendAddressPools/pool1",
			existingBackendPools: []string{
				"/subscriptions/sub/resourceGroups/rg/providers/Microsoft.Network/loadBalancers/lb1-internal/backendAddressPools/pool2",
			},
			expected:       true,
			expectedLBName: "",
		},
		{
			backendPoolID: "/subscriptions/sub/resourceGroups/rg/providers/Microsoft.Network/loadBalancers/lb1/backendAddressPools/pool1",
			existingBackendPools: []string{
				"/subscriptions/sub/resourceGroups/rg/providers/Microsoft.Network/loadBalancers/lb2/backendAddressPools/pool2",
			},
			expected:       false,
			expectedLBName: "lb2",
		},
		{
			backendPoolID: "wrong-backendpool-id",
			existingBackendPools: []string{
				"/subscriptions/sub/resourceGroups/rg/providers/Microsoft.Network/loadBalancers/lb1/backendAddressPools/pool2",
			},
			expectError: true,
		},
		{
			backendPoolID: "/subscriptions/sub/resourceGroups/rg/providers/Microsoft.Network/loadBalancers/lb1/backendAddressPools/pool1",
			existingBackendPools: []string{
				"wrong-existing-backendpool-id",
			},
			expectError: true,
		},
		{
			backendPoolID: "wrong-backendpool-id",
			existingBackendPools: []string{
				"wrong-existing-backendpool-id",
			},
			expectError: true,
		},
		{
			backendPoolID: "/subscriptions/sub/resourceGroups/rg/providers/Microsoft.Network/loadBalancers/malformed-lb1-internal/backendAddressPools/pool1",
			existingBackendPools: []string{
				"/subscriptions/sub/resourceGroups/rg/providers/Microsoft.Network/loadBalancers/malformed-lb1-lanretni/backendAddressPools/pool2",
			},
			expected:       false,
			expectedLBName: "malformed-lb1-lanretni",
		},
	}

	for _, test := range tests {
		isSameLB, lbName, err := isBackendPoolOnSameLB(test.backendPoolID, test.existingBackendPools)
		if test.expectError {
			assert.Error(t, err)
			continue
		}

		assert.Equal(t, test.expected, isSameLB)
		assert.Equal(t, test.expectedLBName, lbName)
	}
}

func TestGetPublicIPAddress(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tests := []struct {
		desc          string
		pipCache      []network.PublicIPAddress
		expectPIPList bool
		existingPIPs  []network.PublicIPAddress
		expectExists  bool
		expectedPIP   network.PublicIPAddress
	}{
		{
			desc:         "getPublicIPAddress should return pip from cache when it exists",
			pipCache:     []network.PublicIPAddress{{Name: pointer.String("pip")}},
			expectExists: true,
			expectedPIP:  network.PublicIPAddress{Name: pointer.String("pip")},
		},
		{
			desc:          "getPublicIPAddress should from list call when cache is empty",
			expectPIPList: true,
			existingPIPs: []network.PublicIPAddress{
				{Name: pointer.String("pip")},
				{Name: pointer.String("pip1")},
			},
			expectExists: true,
			expectedPIP:  network.PublicIPAddress{Name: pointer.String("pip")},
		},
		{
			desc:          "getPublicIPAddress should try listing when pip does not exist",
			pipCache:      []network.PublicIPAddress{{Name: pointer.String("pip1")}},
			expectPIPList: true,
			existingPIPs:  []network.PublicIPAddress{{Name: pointer.String("pip1")}},
			expectExists:  false,
			expectedPIP:   network.PublicIPAddress{},
		},
	}
	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			pipCache := &sync.Map{}
			for _, pip := range test.pipCache {
				pip := pip
				pipCache.Store(pointer.StringDeref(pip.Name, ""), &pip)
			}
			az := GetTestCloud(ctrl)
			az.pipCache.Set(az.ResourceGroup, pipCache)
			mockPIPsClient := az.PublicIPAddressesClient.(*mockpublicipclient.MockInterface)
			if test.expectPIPList {
				mockPIPsClient.EXPECT().List(gomock.Any(), az.ResourceGroup).Return(test.existingPIPs, nil).MaxTimes(2)
			}
			pip, pipExists, err := az.getPublicIPAddress(az.ResourceGroup, "pip", azcache.CacheReadTypeDefault)
			assert.Equal(t, test.expectedPIP, pip)
			assert.Equal(t, test.expectExists, pipExists)
			assert.NoError(t, err)
		})
	}
}

func TestListPIP(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tests := []struct {
		desc          string
		pipCache      []network.PublicIPAddress
		expectPIPList bool
		existingPIPs  []network.PublicIPAddress
	}{
		{
			desc:     "listPIP should return data from cache, when data is empty slice",
			pipCache: []network.PublicIPAddress{},
		},
		{
			desc: "listPIP should return data from cache",
			pipCache: []network.PublicIPAddress{
				{Name: pointer.String("pip1")},
				{Name: pointer.String("pip2")},
			},
		},
		{
			desc:          "listPIP should return data from arm list call",
			expectPIPList: true,
			existingPIPs:  []network.PublicIPAddress{{Name: pointer.String("pip")}},
		},
	}
	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			az := GetTestCloud(ctrl)
			if test.pipCache != nil {
				pipCache := &sync.Map{}
				for _, pip := range test.pipCache {
					pip := pip
					pipCache.Store(pointer.StringDeref(pip.Name, ""), &pip)
				}
				az.pipCache.Set(az.ResourceGroup, pipCache)
			}
			mockPIPsClient := az.PublicIPAddressesClient.(*mockpublicipclient.MockInterface)
			if test.expectPIPList {
				mockPIPsClient.EXPECT().List(gomock.Any(), az.ResourceGroup).Return(test.existingPIPs, nil).MaxTimes(2)
			}
			pips, err := az.listPIP(az.ResourceGroup, azcache.CacheReadTypeDefault)
			if test.expectPIPList {
				assert.ElementsMatch(t, test.existingPIPs, pips)
			} else {
				assert.ElementsMatch(t, test.pipCache, pips)
			}
			assert.NoError(t, err)
		})
	}
}

func TestPIPClientList(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tests := []struct {
		desc         string
		existingPIPs []network.PublicIPAddress
	}{
		{
			desc:         "should return data from ARM list call",
			existingPIPs: []network.PublicIPAddress{{Name: pointer.String("pip")}},
		},
	}
	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			az := GetTestCloud(ctrl)
			mockPIPsClient := az.PublicIPAddressesClient.(*mockpublicipclient.MockInterface)
			mockPIPsClient.EXPECT().List(gomock.Any(), az.ResourceGroup).Return(test.existingPIPs, nil).Times(1)
			pips, err := az.pipClientList(az.ResourceGroup)
			assert.NotNil(t, pips)
			assert.ElementsMatch(t, test.existingPIPs, *pips)
			assert.NoError(t, err)
		})
	}
}

func TestLBClientGet(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tests := []struct {
		desc       string
		existingLB network.LoadBalancer
	}{
		{
			desc:       "should return data from ARM list call",
			existingLB: network.LoadBalancer{Name: pointer.String("lb")},
		},
	}
	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			az := GetTestCloud(ctrl)
			mockLBClient := az.LoadBalancerClient.(*mockloadbalancerclient.MockInterface)
			mockLBClient.EXPECT().Get(gomock.Any(), az.ResourceGroup, "lb", "").Return(test.existingLB, nil).Times(1)
			lb, err := az.lbClientGet("lb")
			assert.NotNil(t, lb)
			assert.Equal(t, test.existingLB, *(lb.(*network.LoadBalancer)))
			assert.NoError(t, err)
		})
	}
}

func TestPLSClientGet(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tests := []struct {
		desc         string
		fipConfigID  string
		existingPLSs []network.PrivateLinkService
	}{
		{
			desc: "should return data from ARM list call",
			existingPLSs: []network.PrivateLinkService{
				{
					Name: pointer.String("pls"),
					PrivateLinkServiceProperties: &network.PrivateLinkServiceProperties{
						LoadBalancerFrontendIPConfigurations: &[]network.FrontendIPConfiguration{
							{ID: pointer.String("fipConfigID")},
						},
					},
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			az := GetTestCloud(ctrl)
			mockPLSClient := az.PrivateLinkServiceClient.(*mockprivatelinkserviceclient.MockInterface)
			mockPLSClient.EXPECT().List(gomock.Any(), az.PrivateLinkServiceResourceGroup).Return(test.existingPLSs, nil).Times(1)
			pls, err := az.plsClientGet("fipConfigID")
			assert.NotNil(t, pls)
			assert.Equal(t, test.existingPLSs[0], *(pls.(*network.PrivateLinkService)))
			assert.NoError(t, err)
		})
	}
}

func TestNSGClientGet(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tests := []struct {
		desc        string
		existingNSG network.SecurityGroup
	}{
		{
			desc:        "should return data from ARM list call",
			existingNSG: network.SecurityGroup{Name: pointer.String("nsg")},
		},
	}
	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			az := GetTestCloud(ctrl)
			mockNSGClient := az.SecurityGroupsClient.(*mocksecuritygroupclient.MockInterface)
			mockNSGClient.EXPECT().Get(gomock.Any(), az.SecurityGroupResourceGroup, "nsg", "").Return(test.existingNSG, nil).Times(1)
			nsg, err := az.nsgClientGet("nsg")
			assert.NotNil(t, nsg)
			assert.Equal(t, test.existingNSG, *(nsg.(*network.SecurityGroup)))
			assert.NoError(t, err)
		})
	}
}

func TestRouteTableClientGet(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tests := []struct {
		desc       string
		existingRT network.RouteTable
	}{
		{
			desc:       "should return data from ARM list call",
			existingRT: network.RouteTable{Name: pointer.String("rt")},
		},
	}
	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			az := GetTestCloud(ctrl)
			mockRTClient := az.RouteTablesClient.(*mockroutetableclient.MockInterface)
			mockRTClient.EXPECT().Get(gomock.Any(), az.RouteTableResourceGroup, "rt", "").Return(test.existingRT, nil).Times(1)
			rt, err := az.routeTableClientGet("rt")
			assert.NotNil(t, rt)
			assert.Equal(t, test.existingRT, *(rt.(*network.RouteTable)))
			assert.NoError(t, err)
		})
	}
}

func TestVMClientGet(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tests := []struct {
		desc        string
		existingVM  compute.VirtualMachine
		returnEmpty bool
	}{
		{
			desc:       "should return data from ARM list call",
			existingVM: compute.VirtualMachine{Name: pointer.String("vm")},
		},
		{
			desc: "VM under deletion",
			existingVM: compute.VirtualMachine{
				Name: pointer.String("vm"),
				VirtualMachineProperties: &compute.VirtualMachineProperties{
					ProvisioningState: pointer.String(consts.ProvisioningStateDeleting),
				},
			},
			returnEmpty: true,
		},
	}
	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			az := GetTestCloud(ctrl)
			mockVMClient := az.VirtualMachinesClient.(*mockvmclient.MockInterface)
			mockVMClient.EXPECT().Get(gomock.Any(), az.ResourceGroup, "vm", compute.InstanceViewTypesInstanceView).Return(test.existingVM, nil).Times(1)
			vm, err := az.vmClientGet("vm")
			if test.returnEmpty {
				assert.Nil(t, vm)
			} else {
				assert.Equal(t, test.existingVM, *(vm.(*compute.VirtualMachine)))
			}
			assert.NoError(t, err)
		})
	}
}
