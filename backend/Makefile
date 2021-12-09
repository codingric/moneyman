
build:
	go build

package:
	docker login
	docker buildx create --use
	docker buildx build --platform linux/arm64 -t codingric/moneyman-backend --push .