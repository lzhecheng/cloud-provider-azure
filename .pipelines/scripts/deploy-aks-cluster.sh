#!/bin/bash

# Copyright 2022 The Kubernetes Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

export KUBERNETES_VERSION="1.24.0"

IMAGE_TAG="$(git rev-parse --short=7 HEAD)"
# export RESOURCEGROUP="aks-ci-$(echo ${RANDOM_NAME} | md5sum | head -c 7)"
export RESOURCEGROUP="${IMAGE_TAG}"
export CLUSTER="aks-cluster"
export AKS_CLUSTER_ID="/subscriptions/${AZURE_SUBSCRIPTION}/resourcegroups/${RESOURCEGROUP}/providers/Microsoft.ContainerService/managedClusters/${CLUSTER}"

get_random_location() {
    local LOCATIONS=("eastus" "eastus2" "southcentralus" "westus2" "westus3" "australiaeast" "southeastasia" "northeurope" "uksouth" "westeurope" "centralus")
    echo "${LOCATIONS[${RANDOM} % ${#LOCATIONS[@]}]}"
}

export AZURE_LOCATION="$(get_random_location)"

echo "Setting up customconfiguration.json"
CUSTOM_CONFIG_PATH=".pipelines/templates/customconfiguration.json"
CCM_NAME="${IMAGE_REGISTRY}/azure-cloud-controller-manager:${IMAGE_TAG}"
CNM_NAME="${IMAGE_REGISTRY}/azure-cloud-node-manager:${IMAGE_TAG}-linux-amd64"
sed -i "s|CUSTOM_CCM_IMAGE|${CCM_NAME}|g" "${CUSTOM_CONFIG_PATH}"
sed -i "s|CUSTOM_CNM_IMAGE|${CNM_NAME}|g" "${CUSTOM_CONFIG_PATH}"
export CONFIG="$(base64 -w 0 ${CUSTOM_CONFIG_PATH})"

az login --service-principal -u "${SP_CLIENT_ID}" -p "${SP_CLIENT_SECRET}" --tenant "${TENANT_ID}"
export TOKEN=$(az account get-access-token -o json | jq -r '.accessToken')

echo "Creating an AKS cluster in resource group ${RESOURCEGROUP}"
az group create --name "${RESOURCEGROUP}" -l "${AZURE_LOCATION}"
curl -i -X PUT -k -H "Authorization: Bearer ${TOKEN}" \
    -H "Content-Type: application/json" \
    -H "AKSHTTPCustomFeatures: Microsoft.ContainerService/EnableCloudControllerManager" \
    "https://management.azure.com${AKS_CLUSTER_ID}?api-version=2022-04-02-preview" \
    -d "{
         \"id\": \"${AKS_CLUSTER_ID}\",
         \"name\": \"${CLUSTER}\",
         \"location\": \"${AZURE_LOCATION}\",
         \"type\": \"Microsoft.ContainerService/ManagedClusters\",
         \"properties\": {
            \"kubernetesVersion\": \"${KUBERNETES_VERSION}\",
            \"dnsPrefix\": \"aks\",
            \"agentPoolProfiles\": [
              {
                \"name\": \"nodepool1\",
                \"count\": 3,
                \"mode\": \"System\",
                \"vmSize\": \"Standard_DS2_v2\",
                \"osType\": \"Linux\"
              }
            ],
            \"servicePrincipalProfile\": {
               \"clientId\": \"${SP_CLIENT_ID}\",
               \"secret\": \"${SP_CLIENT_SECRET}\"
            },
            \"encodedCustomConfiguration\": \"${CONFIG}\"
        }
    }"

az aks get-credentials --resource-group "${RESOURCEGROUP}" --name "${CLUSTER}" -f aks-cluster.kubeconfig
