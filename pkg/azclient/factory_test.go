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
package azclient

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Factory", func() {
	When("config is nil", func() {
		It("should create factory instance without painc - AvailabilitySet", func() {
			factory, err := NewClientFactory(nil, nil, nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(factory).NotTo(BeNil())
			client := factory.GetAvailabilitySetClient()
			Expect(client).NotTo(BeNil())
		})
		It("should create factory instance without painc - Deployment", func() {
			factory, err := NewClientFactory(nil, nil, nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(factory).NotTo(BeNil())
			client := factory.GetDeploymentClient()
			Expect(client).NotTo(BeNil())
		})
		It("should create factory instance without painc - Disk", func() {
			factory, err := NewClientFactory(nil, nil, nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(factory).NotTo(BeNil())
			client := factory.GetDiskClient()
			Expect(client).NotTo(BeNil())
		})
		It("should create factory instance without painc - Interface", func() {
			factory, err := NewClientFactory(nil, nil, nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(factory).NotTo(BeNil())
			client := factory.GetInterfaceClient()
			Expect(client).NotTo(BeNil())
		})
		It("should create factory instance without painc - IPGroup", func() {
			factory, err := NewClientFactory(nil, nil, nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(factory).NotTo(BeNil())
			client := factory.GetIPGroupClient()
			Expect(client).NotTo(BeNil())
		})
		It("should create factory instance without painc - LoadBalancer", func() {
			factory, err := NewClientFactory(nil, nil, nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(factory).NotTo(BeNil())
			client := factory.GetLoadBalancerClient()
			Expect(client).NotTo(BeNil())
		})
		It("should create factory instance without painc - ManagedCluster", func() {
			factory, err := NewClientFactory(nil, nil, nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(factory).NotTo(BeNil())
			client := factory.GetManagedClusterClient()
			Expect(client).NotTo(BeNil())
		})
		It("should create factory instance without painc - PrivateEndpoint", func() {
			factory, err := NewClientFactory(nil, nil, nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(factory).NotTo(BeNil())
			client := factory.GetPrivateEndpointClient()
			Expect(client).NotTo(BeNil())
		})
		It("should create factory instance without painc - PrivateLinkService", func() {
			factory, err := NewClientFactory(nil, nil, nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(factory).NotTo(BeNil())
			client := factory.GetPrivateLinkServiceClient()
			Expect(client).NotTo(BeNil())
		})
		It("should create factory instance without painc - PrivateZone", func() {
			factory, err := NewClientFactory(nil, nil, nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(factory).NotTo(BeNil())
			client := factory.GetPrivateZoneClient()
			Expect(client).NotTo(BeNil())
		})
		It("should create factory instance without painc - PublicIPAddress", func() {
			factory, err := NewClientFactory(nil, nil, nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(factory).NotTo(BeNil())
			client := factory.GetPublicIPAddressClient()
			Expect(client).NotTo(BeNil())
		})
		It("should create factory instance without painc - PublicIPPrefix", func() {
			factory, err := NewClientFactory(nil, nil, nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(factory).NotTo(BeNil())
			client := factory.GetPublicIPPrefixClient()
			Expect(client).NotTo(BeNil())
		})
		It("should create factory instance without painc - Registry", func() {
			factory, err := NewClientFactory(nil, nil, nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(factory).NotTo(BeNil())
			client := factory.GetRegistryClient()
			Expect(client).NotTo(BeNil())
		})
		It("should create factory instance without painc - ResourceGroup", func() {
			factory, err := NewClientFactory(nil, nil, nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(factory).NotTo(BeNil())
			client := factory.GetResourceGroupClient()
			Expect(client).NotTo(BeNil())
		})
		It("should create factory instance without painc - RouteTable", func() {
			factory, err := NewClientFactory(nil, nil, nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(factory).NotTo(BeNil())
			client := factory.GetRouteTableClient()
			Expect(client).NotTo(BeNil())
		})
		It("should create factory instance without painc - SecurityGroup", func() {
			factory, err := NewClientFactory(nil, nil, nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(factory).NotTo(BeNil())
			client := factory.GetSecurityGroupClient()
			Expect(client).NotTo(BeNil())
		})
		It("should create factory instance without painc - Snapshot", func() {
			factory, err := NewClientFactory(nil, nil, nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(factory).NotTo(BeNil())
			client := factory.GetSnapshotClient()
			Expect(client).NotTo(BeNil())
		})
		It("should create factory instance without painc - SSHPublicKeyResource", func() {
			factory, err := NewClientFactory(nil, nil, nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(factory).NotTo(BeNil())
			client := factory.GetSSHPublicKeyResourceClient()
			Expect(client).NotTo(BeNil())
		})
		It("should create factory instance without painc - Subnet", func() {
			factory, err := NewClientFactory(nil, nil, nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(factory).NotTo(BeNil())
			client := factory.GetSubnetClient()
			Expect(client).NotTo(BeNil())
		})
		It("should create factory instance without painc - VirtualMachine", func() {
			factory, err := NewClientFactory(nil, nil, nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(factory).NotTo(BeNil())
			client := factory.GetVirtualMachineClient()
			Expect(client).NotTo(BeNil())
		})
		It("should create factory instance without painc - VirtualMachineScaleSet", func() {
			factory, err := NewClientFactory(nil, nil, nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(factory).NotTo(BeNil())
			client := factory.GetVirtualMachineScaleSetClient()
			Expect(client).NotTo(BeNil())
		})
		It("should create factory instance without painc - VirtualMachineScaleSetVM", func() {
			factory, err := NewClientFactory(nil, nil, nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(factory).NotTo(BeNil())
			client := factory.GetVirtualMachineScaleSetVMClient()
			Expect(client).NotTo(BeNil())
		})
		It("should create factory instance without painc - VirtualNetwork", func() {
			factory, err := NewClientFactory(nil, nil, nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(factory).NotTo(BeNil())
			client := factory.GetVirtualNetworkClient()
			Expect(client).NotTo(BeNil())
		})
	})
})