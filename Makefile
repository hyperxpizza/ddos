build:
	go build -o bin/ main.go
docker_build:
	docker build --tag golang-ddos .
docker_run:
	docker run -d golang-ddos