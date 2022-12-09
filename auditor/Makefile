include ../Makefile.common

.PHONY: kustomize

kustomize:
	echo "Connecting from $(shell curl -s https://ifconfig.me/ip)" \
	&& echo GIT_SHA=$(GIT_SHA) > kustomize/deploy.env \
	&& echo DEPLOYMENT_TIME=$(shell date +%s) >> kustomize/deploy.env \
	&& kubectl apply -k kustomize