
build:
	go build

package:
	docker buildx build \
		--platform linux/arm64 \
		-t ghcr.io/codingric/moneyman-bigbills \
		--progress plain \
		--push .