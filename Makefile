all:
	@echo "use release target to build release binary"

generate:
	@go generate ./...

clean:
	@echo "removing binary and profiling files"
	@rm -f gopher2600 cpu.profile mem.profile

release:
	go build -gcflags '-c 3 -B -+ -wb=false' .

