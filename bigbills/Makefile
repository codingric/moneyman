
build:
	go build

package:
	docker buildx build --platform linux/amd64,linux/arm64,linux/arm/v7 -t ghcr.io/codingric/moneyman-bigbills --push .