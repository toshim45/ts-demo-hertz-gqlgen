help:
	@echo "HELP: make clean|build|run"
clean:
	@echo "cleaning"
	rm -rfv binary
build: clean
	@echo "building"
	go mod tidy
	go build -o binary server.go
run:
	@echo "running"
	./binary
