.PHONY: build-controller-image
build-controller-image:
	docker build --tag pgbackrest-controller:latest --target pgbackrest-controller \
		-f containers/pgbackrestPlugin.containers .

.PHONY: build-sidecar-image
build-sidecar-image:
	docker build --tag pgbackrest-sidecar:latest --target pgbackrest-sidecar \
		-f containers/pgbackrestPlugin.containers .

.PHONY: build-images
build-images: build-sidecar-image build-controller-image

.PHONY: create-kind-e2e-cluster
create-kind-e2e-cluster:
	kind create cluster --name e2e-cnpg-pgbackrest

.PHONY: images-upload-kind-e2e-cluster
upload-images-kind-e2e-cluster: build-images
	kind load docker-image pgbackrest-controller:latest --name e2e-cnpg-pgbackrest
	kind load docker-image pgbackrest-sidecar:latest --name e2e-cnpg-pgbackrest

.PHONY: test-e2e-run-tests
test-e2e-run-tests:
	go test -v test/e2e/e2e_test.go

.PHONY: test-e2e
test-e2e: create-kind-e2e-cluster upload-images-kind-e2e-cluster test-e2e-run-tests

.PHONY: clean-kind-e2e-cluster
clean-kind-e2e-cluster:
	kind delete cluster --name e2e-cnpg-pgbackrest
