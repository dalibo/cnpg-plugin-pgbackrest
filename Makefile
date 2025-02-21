.PHONY: run
run: image


.PHONY: image
controller:
	docker build -t pgbackrest-plugin:latest -f containers/Operator.container  .

sidecar:
	docker build -t pgbackrest-sidecar:latest -f containers/Sidecar.container  .

images: sidecar controller

images-upload-kind: images
	kind load docker-image pgbackrest-plugin:latest --name pg-operator-e2e-v1-31-2
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

