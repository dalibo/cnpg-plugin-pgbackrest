.PHONY: run
run: image


.PHONY: image
controller:
	docker build --tag pgbackrest-controller:latest --target pgbackrest-controller \
		-f containers/pgbackrestPlugin.containers .

sidecar:
	docker build --tag pgbackrest-sidecar:latest --target pgbackrest-sidecar \
		-f containers/pgbackrestPlugin.containers .

images: sidecar controller

images-upload-kind: images
	kind load docker-image pgbackrest-controller:latest --name pg-operator-e2e-v1-31-2
	kind load docker-image pgbackrest-sidecar:latest --name pg-operator-e2e-v1-31-2

deploy:
	kubectl apply -f ./kubernetes

test: deploy
	kubectl apply -f instance.yml

deploy-kind: images-upload-kind deploy

clean:
	kubectl delete -f instance.yml
	sleep 3
	kubectl delete -f ./kubernetes

