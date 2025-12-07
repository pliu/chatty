.PHONY: run test build clean

LOCAL_IP := $(shell ifconfig | grep "inet " | grep -Fv 127.0.0.1 | awk '{print $$2}' | head -n1)

run:
	@echo "Running on https://$(LOCAL_IP):8443"
	cd go && go run main.go -base-url=https://$(LOCAL_IP):8443

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
