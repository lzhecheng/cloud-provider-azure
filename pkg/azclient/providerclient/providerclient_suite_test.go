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
package providerclient

import (
	"context"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"sigs.k8s.io/cloud-provider-azure/pkg/azclient/recording"
)

func TestClient(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Client Suite")
}

var resourceGroupName = "aks-cit-Provider"
var resourceName = "testResource"

var subscriptionID string
var location = "eastus"
var resourceGroupClient *armresources.ResourceGroupsClient
var err error
var recorder *recording.Recorder
var realClient Interface

var _ = BeforeSuite(func(ctx context.Context) {
	recorder, err = recording.NewRecorder("testdata/Provider")
	Expect(err).ToNot(HaveOccurred())
	subscriptionID = recorder.SubscriptionID()
	Expect(err).NotTo(HaveOccurred())
	cred := recorder.TokenCredential()
	resourceGroupClient, err = armresources.NewResourceGroupsClient(subscriptionID, cred, &arm.ClientOptions{
		ClientOptions: azcore.ClientOptions{
			Transport: recorder.HTTPClient(),
		},
	})
	Expect(err).NotTo(HaveOccurred())
	realClient, err = New(subscriptionID, recorder.TokenCredential(), &arm.ClientOptions{
		ClientOptions: azcore.ClientOptions{
			Transport: recorder.HTTPClient(),
		},
	})
	Expect(err).NotTo(HaveOccurred())
	_, err = resourceGroupClient.CreateOrUpdate(
		ctx,
		resourceGroupName,
		armresources.ResourceGroup{
			Location: to.Ptr(location),
		},
		nil)
	Expect(err).NotTo(HaveOccurred())
})

var _ = AfterSuite(func(ctx context.Context) {
	_, err := resourceGroupClient.BeginDelete(ctx, resourceGroupName, nil)
	Expect(err).NotTo(HaveOccurred())

	err = recorder.Stop()
	Expect(err).ToNot(HaveOccurred())
})
