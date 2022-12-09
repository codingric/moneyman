include ../Makefile.common

update-config:
	@export SECRET_YAML=$(shell mktemp) CONFIG_YAML=$(shell mktemp) && \
	sops -d kustomize/auditor-secret.enc.yaml > $$SECRET_YAML && \
	yq e '.data."config.yaml"' $$SECRET_YAML | base64 -d > $$CONFIG_YAML && \
	vim -f $$CONFIG_YAML && \
	export DATA=`base64 $$CONFIG_YAML` && \
	yq e '.data."config.yaml" = env(DATA)' -i $$SECRET_YAML && \
	sops -e --input-type yaml --output-type yaml $$SECRET_YAML > kustomize/auditor-secret.enc.yaml

kustomize:
	echo GIT_SHA=$(GIT_SHA) > kustomize/deploy.env; \
	echo DEPLOYMENT_TIME=$(shell date +%s) >> kustomize/deploy.env; \
	curl -s https://ifconfig.me/ip
	kubectl apply -k kustomize