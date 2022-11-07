set -o errexit
set -o nounset
set -o pipefail

REPO_ROOT=$(dirname "${BASH_SOURCE[0]}")/..

export CLUSTER_TYPE="test-type"
go get -d sigs.k8s.io/kubetest2@latest
go install sigs.k8s.io/kubetest2@latest

# curl -sL https://aka.ms/InstallAzureCLIDeb | sudo bash
"${REPO_ROOT}"/.pipelines/scripts/run-e2e.sh
