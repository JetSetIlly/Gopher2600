compileFlags = '-c 3 -B -wb=false'
profilingRom = roms/Pitfall.bin

.PHONY: all generate test clean run_assertions race build_assertions build release check_upx release_upx profile profile_display

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
	@rm -f gopher2600.exe
	@find ./ -type f | grep "\.orig" | xargs -r rm

race:
	go run -race -gcflags=all=-d=checkptr=0 gopher2600.go debug roms/Homebrew/chaoticGrill-2019-08-18--NTSC.bin

build_assertions:
	go build -gcflags $(compileFlags) -tags=assertions

build:
	go build -gcflags $(compileFlags)

release:
	go build -gcflags $(compileFlags) -ldflags="-s -w" -tags="release"

check_upx:
	@which upx > /dev/null

release_upx: check_upx
	go build -gcflags $(compileFlags) -ldflags="-s -w" -tags="release"
	upx -o gopher2600.upx gopher2600
	cp gopher2600.upx gopher2600
	rm gopher2600.upx

profile:
	go build -gcflags $(compileFlags)
	./gopher2600 performance --profile $(profilingRom)
	go tool pprof -http : ./gopher2600 cpu.profile

profile_display:
	go build -gcflags $(compileFlags)
	./gopher2600 performance --display --profile $(profilingRom)
	go tool pprof -http : ./gopher2600 cpu.profile

cross_windows:
	CGO_ENABLED="1" CC="/usr/bin/x86_64-w64-mingw32-gcc" CXX="/usr/bin/x86_64-w64-mingw32-g++" GOOS="windows" GOARCH="amd64" CGO_LDFLAGS="-lmingw32 -lSDL2" CGO_CFLAGS="-D_REENTRANT" go build -tags "release" -ldflags="-s -w" .

cross_windows_static:
	CGO_ENABLED="1" CC="/usr/bin/x86_64-w64-mingw32-gcc" CXX="/usr/bin/x86_64-w64-mingw32-g++" GOOS="windows" GOARCH="amd64" CGO_LDFLAGS="-static-libgcc -static-libstdc++" go build -tags "static release" -ldflags "-s -w" .
