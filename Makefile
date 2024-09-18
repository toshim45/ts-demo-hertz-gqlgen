help:
	@echo "HELP: make clean|gqlgen|build|run"
clean:
	@echo "cleaning"
	rm -rfv binary
gqlgen:
	go run github.com/99designs/gqlgen generate
build: clean
	@echo "building"
	go mod tidy
	go build -o binary server.go
run:
	@echo "running"
	./binary
