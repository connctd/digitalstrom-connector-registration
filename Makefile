default: build


run:
	make build
	./ds-connector-registration

build:
	go build

windows:
	env GOOS=windows go build

linux:
	env GOOS=linux go build

macos:
	env GOOS=darwin go build