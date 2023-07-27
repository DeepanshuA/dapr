# Set the Azure region to use (needs to support Availability Zones)
AZURE_REGION="westus2"

# Name prefix for your test resources
# Try to use a unique name，at least 4 characters
export TEST_PREFIX="mydapraks007"

# Set to true to add a Windows node pool to the AKS cluster
ENABLE_WINDOWS="false"

# Name of the resource group where to deploy your cluster
export TEST_RESOURCE_GROUP="deeaga_rg001"

# Create a resource group
az group create \
  --resource-group "${TEST_RESOURCE_GROUP}" \
  --location "${AZURE_REGION}"

# Deploy the test infrastructure
az deployment group create \
  --resource-group "${TEST_RESOURCE_GROUP}" \
  --template-file ./tests/test-infra/azure-aks.bicep \
  --parameters namePrefix=${TEST_PREFIX} location=${AZURE_REGION} enableWindows=${ENABLE_WINDOWS}

# Authenticate with Azure Container Registry
az acr login --name "${TEST_PREFIX}acr"

# Connect to AKS
az aks get-credentials -n "${TEST_PREFIX}-aks" -g "${TEST_RESOURCE_GROUP}"

# Set the value for DAPR_REGISTRY
export DAPR_REGISTRY="${TEST_PREFIX}acr.azurecr.io"

# Set the value for DAPR_NAMESPACE as per instructions above and create the namespace
export DAPR_NAMESPACE=dapr-tests
make create-test-namespace
