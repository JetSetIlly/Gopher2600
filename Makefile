compileFlags = '-c 3 -B -wb=false'
profilingRom = roms/Pitfall.bin

.PHONY: all generate test clean run_assertions race vet build_assertions build release check_upx release_upx profile profile_display

all:
	@echo "use release target to build release binary"

clean:
	@echo "removing binary and profiling files"
	@rm -f gopher2600 cpu.profile mem.profile debug.cpu.profile debug.mem.profile
	@rm -f gopher2600.exe
	@find ./ -type f | grep "\.orig" | xargs -r rm

tidy:
	goimports -w .

generate:
	@go generate ./...

check_lint:
ifeq (, $(shell which golangci-lint))
	$(error not golanci-lint not installed)
endif

lint: check_lint
# uses .golangci.yml configuration file
	golangci-lint run --sort-results

test:
	go test -tags=testing ./...

race: generate lint vet test
# disable checkptr because the opengl implementation will trigger it and cause
# a lot of output noise
	go run -race -gcflags=all=-d=checkptr=0 gopher2600.go debug roms/Pitfall.bin
#go run -race -gcflags=all=-d=checkptr=0 gopher2600.go debug "roms/starpath/02 - Communist Mutants From Space (Ntsc).mp3"

profile: generate lint vet test
	go build -gcflags $(compileFlags)
	./gopher2600 performance --profile $(profilingRom)
	go tool pprof -http : ./gopher2600 cpu.profile

profile_display: generate lint vet test
	go build -gcflags $(compileFlags)
	./gopher2600 performance --display --profile $(profilingRom)
	go tool pprof -http : ./gopher2600 cpu.profile

build_assertions: generate lint vet test
	go build -gcflags $(compileFlags) -tags=assertions

build: generate lint vet test
	go build -gcflags $(compileFlags)

release: generate lint vet test
	go build -gcflags $(compileFlags) -ldflags="-s -w" -tags="release"

check_upx:
ifeq (, $(shell which upx))
	$(error upx not installed")
endif

release_upx: check_upx generate lint vet test
	go build -gcflags $(compileFlags) -ldflags="-s -w" -tags="release"
	upx -o gopher2600.upx gopher2600
	cp gopher2600.upx gopher2600
	rm gopher2600.upx

cross_windows: generate lint vet test
	CGO_ENABLED="1" CC="/usr/bin/x86_64-w64-mingw32-gcc" CXX="/usr/bin/x86_64-w64-mingw32-g++" GOOS="windows" GOARCH="amd64" CGO_LDFLAGS="-lmingw32 -lSDL2" CGO_CFLAGS="-D_REENTRANT" go build -tags "release" -ldflags="-s -w" .

cross_windows_static: generate lint vet test
	CGO_ENABLED="1" CC="/usr/bin/x86_64-w64-mingw32-gcc" CXX="/usr/bin/x86_64-w64-mingw32-g++" GOOS="windows" GOARCH="amd64" CGO_LDFLAGS="-static-libgcc -static-libstdc++" go build -tags "static release" -ldflags "-s -w" .
