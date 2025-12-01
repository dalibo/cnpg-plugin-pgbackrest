# SPDX-FileCopyrightText: 2025 Dalibo
#
# SPDX-License-Identifier: Apache-2.0

##@ General

# The help target prints out all targets with their descriptions organized
# beneath their categories. The categories are represented by '##@' and the
# target descriptions by '##'. The awk command is responsible for reading the
# entire set of makefiles included in this invocation, looking for lines of the
# file as xyz: ## something, and then pretty-format the target and help. Then,
# if there's a line with ##@ something, that gets pretty-printed as a category.
# More info on the usage of ANSI control characters for terminal formatting:
# https://en.wikipedia.org/wiki/ANSI_escape_code#SGR_parameters
# More info on the awk command:
# http://linuxcommand.org/lc3_adv_awk.php

KIND ?= kind
CONTAINER_TOOL ?= docker
KIND_CLUSTER ?= e2e-cnpg-pgbackrest
CONTROLLER_GEN ?= controller-gen

.PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Development

.PHONY: manifests
manifests: ## Generate WebhookConfiguration, ClusterRole and CustomResourceDefinition objects.
	$(CONTROLLER_GEN) rbac:roleName=pgbackrest-controller crd webhook paths="./internal/pgbackrest/..." paths="./api/..." output:crd:artifacts:config=config/crd/bases

.PHONY: generate
generate: ## Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations.
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./internal/pgbackrest/api/..."

.PHONY: fmt
fmt: ## Run go fmt against code.
	go fmt ./...

.PHONY: vet
vet: ## Run go vet against code.
	go vet ./...


.PHONY: setup-test-e2e
setup-test-e2e: ## Set up a Kind cluster for e2e tests if it does not exist
	@command -v $(KIND) >/dev/null 2>&1 || { \
		echo "Kind is not installed. Please install Kind manually."; \
		exit 1; \
	}
	@case "$$($(KIND) get clusters)" in \
		*"$(KIND_CLUSTER)"*) \
			echo "Kind cluster '$(KIND_CLUSTER)' already exists. Skipping creation." ;; \
		*) \
			echo "Creating Kind cluster '$(KIND_CLUSTER)'..."; \
			$(KIND) create cluster --name $(KIND_CLUSTER) ;; \
	esac

.PHONY: images-upload-kind-e2e-cluster
upload-images-kind-e2e-cluster: build-images ## Load container images to the K8S test cluster
	@$(KIND) load docker-image pgbackrest-controller:latest --name $(KIND_CLUSTER)
	@$(KIND) load docker-image pgbackrest-sidecar:latest --name $(KIND_CLUSTER)

.PHONY: test-e2e-run-tests
test-e2e-run-tests:
	go test -v test/e2e/e2e_test.go

.PHONY: test-e2e
test-e2e: setup-test-e2e upload-images-kind-e2e-cluster test-e2e-run-tests ## Run the e2e tests.

.PHONY: cleanup-test-e2e
cleanup-test-e2e: ## Tear down the Kind cluster used for e2e tests
	@$(KIND) delete cluster --name $(KIND_CLUSTER)

##@ Build

.PHONY: build-controller-image
build-controller-image: ## Build controller image.
	$(CONTAINER_TOOL) build --tag pgbackrest-controller:latest \
		-f containers/pgbackrest-controller.container .

.PHONY: build-sidecar-image
build-sidecar-image: ## Build sidecar image
	$(CONTAINER_TOOL) build --tag pgbackrest-sidecar:latest \
		-f containers/pgbackrest-sidecar.container .

.PHONY: build-images
build-images: build-sidecar-image build-controller-image ## Build controller and sidecar images.
