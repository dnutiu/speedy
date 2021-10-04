.PHONY: docker-build
docker-build:
	docker build . -f ./Dockerfile -t speedy

.PHONY: docker-build-custom-kafka
docker-build-custom-kafka:
	docker build . -f ./Dockerfile.librdkafka -t speedy_kafka