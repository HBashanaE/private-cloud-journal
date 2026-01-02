run-dev:
	go run cmd/private-cloud-journal/main.go

build:
	mkdir -p build
	go build -o build/PrivateCloudJournal cmd/private-cloud-journal/main.go
clean:
	rm -rf build
run:
	./build/PrivateCloudJournal
