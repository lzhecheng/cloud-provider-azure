// /*
// Copyright The Kubernetes Authors.
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
// */

// Code generated by client-gen. DO NOT EDIT.
package privatelinkserviceclient

import (
	"context"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/runtime"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	armnetwork "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v3"
	. "github.com/onsi/gomega"
)

var (
	networkClientFactory *armnetwork.ClientFactory
	loadBalancersClient  *armnetwork.LoadBalancersClient
	pipClient            *armnetwork.PublicIPAddressesClient
	vnetClient           *armnetwork.VirtualNetworksClient
)
var (
	pipResource  *armnetwork.PublicIPAddress
	lbResource   *armnetwork.LoadBalancer
	vnetResource *armnetwork.VirtualNetwork
)

func init() {
	addtionalTestCases = func() {
	}

	beforeAllFunc = func(ctx context.Context) {
		networkClientFactory, err = armnetwork.NewClientFactory(subscriptionID, recorder.TokenCredential(), &arm.ClientOptions{
			ClientOptions: policy.ClientOptions{
				Transport: recorder.HTTPClient(),
			},
		})
		Expect(err).NotTo(HaveOccurred())

		pipClient = networkClientFactory.NewPublicIPAddressesClient()
		poller, err := pipClient.BeginCreateOrUpdate(ctx, resourceGroupName, "pip1", armnetwork.PublicIPAddress{
			Location: to.Ptr(location),
			Properties: &armnetwork.PublicIPAddressPropertiesFormat{
				PublicIPAddressVersion:   to.Ptr(armnetwork.IPVersionIPv4),
				PublicIPAllocationMethod: to.Ptr(armnetwork.IPAllocationMethodStatic),
			},
			SKU: &armnetwork.PublicIPAddressSKU{
				Name: to.Ptr(armnetwork.PublicIPAddressSKUNameStandard),
				Tier: to.Ptr(armnetwork.PublicIPAddressSKUTierRegional),
			},
		}, nil)
		Expect(err).NotTo(HaveOccurred())
		resp, err := poller.PollUntilDone(ctx, &runtime.PollUntilDoneOptions{Frequency: 1 * time.Second})
		Expect(err).NotTo(HaveOccurred())
		pipResource = &resp.PublicIPAddress

		loadBalancersClient = networkClientFactory.NewLoadBalancersClient()
		lbpoller, err := loadBalancersClient.BeginCreateOrUpdate(ctx, resourceGroupName, "lb1", armnetwork.LoadBalancer{
			Location: to.Ptr(location),
			Properties: &armnetwork.LoadBalancerPropertiesFormat{
				FrontendIPConfigurations: []*armnetwork.FrontendIPConfiguration{
					{
						Name: to.Ptr("frontendConfig1"),
						Properties: &armnetwork.FrontendIPConfigurationPropertiesFormat{
							PublicIPAddress: pipResource,
						},
					},
				},
			},
			SKU: &armnetwork.LoadBalancerSKU{
				Name: to.Ptr(armnetwork.LoadBalancerSKUNameStandard),
				Tier: to.Ptr(armnetwork.LoadBalancerSKUTierRegional),
			},
		}, nil)
		Expect(err).NotTo(HaveOccurred())
		lbresp, err := lbpoller.PollUntilDone(ctx, &runtime.PollUntilDoneOptions{Frequency: 1 * time.Second})
		Expect(err).NotTo(HaveOccurred())
		lbResource = &lbresp.LoadBalancer

		vnetClient = networkClientFactory.NewVirtualNetworksClient()
		vnetpoller, err := vnetClient.BeginCreateOrUpdate(ctx, resourceGroupName, "vnet1", armnetwork.VirtualNetwork{
			Location: to.Ptr(location),
			Properties: &armnetwork.VirtualNetworkPropertiesFormat{
				AddressSpace: &armnetwork.AddressSpace{
					AddressPrefixes: []*string{
						to.Ptr("10.1.0.0/16"),
					},
				},
				Subnets: []*armnetwork.Subnet{
					{
						Name: to.Ptr("subnet1"),
						Properties: &armnetwork.SubnetPropertiesFormat{
							AddressPrefix:                     to.Ptr("10.1.0.0/24"),
							PrivateEndpointNetworkPolicies:    to.Ptr(armnetwork.VirtualNetworkPrivateEndpointNetworkPoliciesDisabled),
							PrivateLinkServiceNetworkPolicies: to.Ptr(armnetwork.VirtualNetworkPrivateLinkServiceNetworkPoliciesDisabled),
						},
					},
				},
			},
		}, nil)
		Expect(err).NotTo(HaveOccurred())
		vnetresp, err := vnetpoller.PollUntilDone(ctx, &runtime.PollUntilDoneOptions{Frequency: 1 * time.Second})
		Expect(err).NotTo(HaveOccurred())
		vnetResource = &vnetresp.VirtualNetwork

		newResource = &armnetwork.PrivateLinkService{
			Location: to.Ptr(location),
			Properties: &armnetwork.PrivateLinkServiceProperties{
				IPConfigurations: []*armnetwork.PrivateLinkServiceIPConfiguration{
					{
						Name: to.Ptr("ipConfig1"),
						Properties: &armnetwork.PrivateLinkServiceIPConfigurationProperties{
							Subnet:                  vnetResource.Properties.Subnets[0],
							Primary:                 to.Ptr(true),
							PrivateIPAddressVersion: to.Ptr(armnetwork.IPVersionIPv4),
						},
					},
				},
				LoadBalancerFrontendIPConfigurations: lbResource.Properties.FrontendIPConfigurations,
			},
		}
	}
	afterAllFunc = func(ctx context.Context) {
		lbPoller, err := loadBalancersClient.BeginDelete(ctx, resourceGroupName, "lb1", nil)
		Expect(err).NotTo(HaveOccurred())
		_, err = lbPoller.PollUntilDone(ctx, &runtime.PollUntilDoneOptions{Frequency: 1 * time.Second})
		Expect(err).NotTo(HaveOccurred())

		pipPoller, err := pipClient.BeginDelete(ctx, resourceGroupName, "pip1", nil)
		Expect(err).NotTo(HaveOccurred())
		_, err = pipPoller.PollUntilDone(ctx, &runtime.PollUntilDoneOptions{Frequency: 1 * time.Second})
		Expect(err).NotTo(HaveOccurred())

		vnetPoller, err := vnetClient.BeginDelete(ctx, resourceGroupName, "vnet1", nil)
		Expect(err).NotTo(HaveOccurred())
		_, err = vnetPoller.PollUntilDone(ctx, &runtime.PollUntilDoneOptions{Frequency: 1 * time.Second})
		Expect(err).NotTo(HaveOccurred())
	}
}
