include ../Makefile.common

kustomize:
	echo GIT_SHA=$(GIT_SHA) > kustomize/deploy.env; \
	echo DEPLOYMENT_TIME=$(shell date +%s) >> kustomize/deploy.env; \
	curl -s https://ifconfig.me/ip
	kubectl apply -k kustomize