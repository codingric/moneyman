
build:
	go build

package:
	docker login
	docker buildx create --use
	docker buildx build --platform linux/amd64,linux/arm64,linux/arm/v7 -t codingric/bigbills --push .