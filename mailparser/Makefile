include ../Makefile.common

test:
	@sed -i .backup '/fixtures/d' .dockerignore; \
	docker-compose run test; \
	export DOCKER_EXITCODE=$$?; \
	mv .dockerignore.backup .dockerignore; \
	exit $$DOCKER_EXITCODE
