
build:
	go build

package:
	docker buildx build \
		--cache-from type=registry,ref=ghcr.io/codingric/moneyman/backend \
		--platform linux/arm64 \
		-t ghcr.io/codingric/moneyman/backend \
		--push .