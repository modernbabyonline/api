up:
	docker-compose up
build: 
	docker-compose up --build
test: 
	go test -v ./...
seedme:
	cd seed && go build . && ./seed
