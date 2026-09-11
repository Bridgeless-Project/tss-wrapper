.PHONY: install

gen-proto:
	cd proto && buf generate

docker-up:
	@echo "Starting Docker Compose..."
	docker-compose up
