all:
	@echo "use release target to build release binary"

generate:
	@go generate ./...

clean:
	@echo "removing binary and profiling files"
	@rm -f web2600.wasm cpu.profile mem.profile

release:
	GOOS=js GOARCH=wasm go build -o ./www/web2600.wasm ./src/

webserve:
	@go run ./test
