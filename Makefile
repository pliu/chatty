.PHONY: run test build clean

run:
	cd go && go run main.go

test:
	cd go && go test -count=1 ./...

build:
	cd go && go build -o ../bin/chatty main.go

clean:
	rm -rf bin/

docker-up:
	docker-compose up -d

docker-down:
	docker-compose down
