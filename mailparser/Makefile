
build:
	go build

package:
	docker buildx create --use
	docker buildx build \
		--cache-from type=registry,ref=ghcr.io/codingric/moneyman/mailparser \
		--platform linux/arm64 \
		-t ghcr.io/codingric/moneyman/mailparser \
		--push .