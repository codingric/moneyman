include ../Makefile.common
RUN = $(shell [ ! -z "$$TEST" ] && echo "-run $$TEST")

update-config:
	@export SECRET_YAML=$(shell mktemp) CONFIG_YAML=$(shell mktemp) && \
	sops -d kustomize/bigbills-secret.enc.yaml > $$SECRET_YAML && \
	yq e '.data."config.yaml"' $$SECRET_YAML | base64 -d > $$CONFIG_YAML && \
	vim -f $$CONFIG_YAML && \
	export DATA=`base64 $$CONFIG_YAML` && \
	yq e '.data."config.yaml" = env(DATA)' -i $$SECRET_YAML && \
	sops -e --input-type yaml --output-type yaml $$SECRET_YAML > kustomize/bigbills-secret.enc.yaml

local-tests:
	@go test ./... -gcflags=all=-l -v -coverprofile coverage.out $(RUN) && go tool cover -html coverage.out

test:
	docker run -v $(shell pwd):/build ghcr.io/codingric/gotester

package:
	cp build.dockerignore .dockerignore || true ; \
	unset DOCKER_HOST ;\
	docker context ls 
	