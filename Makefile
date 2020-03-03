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
	@find ./ -type f | grep "\.orig" | xargs -r rm

build_assertions:
	go build -gcflags $(compileFlags) -tags=assertions

build:
	go build -gcflags $(compileFlags)

release:
	go build -gcflags $(compileFlags) -ldflags="-s -w"

release_upx:
	@echo "requires upx to run. edit Makefile to activate"
	# go build -gcflags $(compileFlags) -ldflags="-s -w"
	# upx -o gopher2600.upx gopher2600
	# cp gopher2600.upx gopher2600
	# rm gopher2600.upx

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
