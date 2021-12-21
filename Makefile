build:
	go build

package:
	docker buildx build \
		--cache-from=type=local,src=/tmp/cache \
		--cache-to=type=local,dest=/tmp/cache \
		--platform linux/arm64 \
		-t ghcr.io/codingric/moneyman/$(IMAGE_NAME):latest \
		--progress plain \
		--push \
		.
