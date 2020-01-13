compileFlags = '-c 3 -B -wb=false'
profilingRom = roms/Pitfall.bin

all:
	@echo "use release target to build release binary"

generate:
	@go generate ./...

gotest:
	go test `go list ./... | grep -v /web2600/)`
	GOOS=js GOARCH=wasm go test ./web2600/...

clean:
	@echo "removing binary and profiling files"
	@rm -f gopher2600 cpu.profile mem.profile debug.cpu.profile debug.mem.profile

build:
	go build -gcflags $(compileFlags)

release:
	@#go build -gcflags $(compileFlags) -tags release
	@echo "use 'make build' for now. the release target will"
	@echo "reappear in a future commit."

profile:
	go build -gcflags $(compileFlags)
	./gopher2600 performance --profile $(profilingRom)
	go tool pprof -http : ./gopher2600 cpu.profile

profile_display:
	go build -gcflags $(compileFlags)
	./gopher2600 performance --display --profile $(profilingRom)
	go tool pprof -http : ./gopher2600 cpu.profile

web:
	cd web2600 && make release && make webserve
