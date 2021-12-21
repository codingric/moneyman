
build:
	go build

package:
	docker buildx build \
		--platform linux/arm64 \
		-t ghcr.io/codingric/moneyman/bigbills \
		-t bigbills \
		--progress plain \
		.
	docker push ghcr.io/codingric/moneyman/bigbills