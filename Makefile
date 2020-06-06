compileFlags = '-c 3 -B -wb=false'
profilingRom = roms/Pitfall.bin

.PHONY: all generate test clean run_assertions race build_assertions build release release_upx profile profile_display

all:
	@echo "use release target to build release binary"

generate:
	@go generate ./...

test:
	go test `go list ./... | grep -v /web2600/)`
	GOOS=js GOARCH=wasm go test ./web2600/...

clean:
	@echo "removing binary and profiling files"
	@rm -f gopher2600 cpu.profile mem.profile debug.cpu.profile debug.mem.profile
	@find ./ -type f | grep "\.orig" | xargs -r rm

race:
	go run -race -gcflags=all=-d=checkptr=0 gopher2600.go debug roms/Pitfall2.bin

build_assertions:
	go build -gcflags $(compileFlags) -tags=assertions

build:
	go build -gcflags $(compileFlags)

release:
	go build -gcflags $(compileFlags) -ldflags="-s -w" -tags="release"

release_upx:
	@echo "requires upx to run. edit Makefile to activate"
	# go build -gcflags $(compileFlags) -ldflags="-s -w" -tags="release"
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
