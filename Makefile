osx:
	GOARCH=amd64 GOOS=darwin go build -v -o build/od2kproxy .

linux64:
	GOARCH=amd64 GOOS=linux go build -v -o build/od2kproxy .

install:
	glide up

tests:
	go test `glide novendor` -v
