BINARY_NAME=vfio-device-plugin

build:
	GOARCH=amd64 GOOS=linux CGO_ENABLED=0 go build -o ${BINARY_NAME} cmd/*

run: build
	./${BINARY_NAME}

clean:
	go clean
	rm -f ${BINARY_NAME}

dep:
	go mod download

image: clean build
	docker build --platform=linux/amd64 -t pic-device-plugin .
