
build:
	go build

package:
	docker buildx build \
		--platform linux/arm64 \
		-t ghcr.io/codingric/moneyman/backend \
		-t backend \
		--progress plain \
		.
	docker push ghcr.io/codingric/moneyman/backend