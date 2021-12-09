
build:
	go build

package:
	docker buildx create --use
	docker buildx build \
		--cache-from type=registry,reghcr.io/codingric/moneyman/backend \
		--platform linux/amd64,linux/arm64,linux/arm/v7 \
		-t ghcr.io/codingric/moneyman/backend \
		--push .