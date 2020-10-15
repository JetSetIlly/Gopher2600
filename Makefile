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

lint:
	golangci-lint run -D govet -D errcheck -D ineffassign -D staticcheck \
		-E bodyclose -E unconvert

vet: 
# filter out expected warnings that we are not worried about: 
# . unkeyed fields when creating imgui.Vec4{} and imgui.Vec2{}
# . "misuse" of unsafe.Pointer is required by opengl implementation
# . the files in which we expect the errors to appear
#
# the awk command returns the number of filtered lines as an exit code. this
# will be returned as an exit code by the make command

	@go vet ./... 2>&1 | \
grep -v "github.com/inkyblackness/imgui-go/v2.Vec[2,4] composite literal uses unkeyed fields" | \
grep -v "gui/sdlimgui/glsl.go.*possible misuse of unsafe.Pointer" | \
grep -v "\# github.com/jetsetilly/gopher2600/gui/sdlimgui" | \
awk 'END{exit NR}'

test:
	go test -tags=testing ./...

race: generate lint vet test
# disable checkptr because the opengl implementation will trigger it and cause
# a lot of output noise
	go run -race -gcflags=all=-d=checkptr=0 gopher2600.go debug roms/Pitfall.bin

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
	@which upx > /dev/null

release_upx: check_upx generate lint vet test
	go build -gcflags $(compileFlags) -ldflags="-s -w" -tags="release"
	upx -o gopher2600.upx gopher2600
	cp gopher2600.upx gopher2600
	rm gopher2600.upx

cross_windows: generate lint vet test
	CGO_ENABLED="1" CC="/usr/bin/x86_64-w64-mingw32-gcc" CXX="/usr/bin/x86_64-w64-mingw32-g++" GOOS="windows" GOARCH="amd64" CGO_LDFLAGS="-lmingw32 -lSDL2" CGO_CFLAGS="-D_REENTRANT" go build -tags "release" -ldflags="-s -w" .

cross_windows_static: generate lint vet test
	CGO_ENABLED="1" CC="/usr/bin/x86_64-w64-mingw32-gcc" CXX="/usr/bin/x86_64-w64-mingw32-g++" GOOS="windows" GOARCH="amd64" CGO_LDFLAGS="-static-libgcc -static-libstdc++" go build -tags "static release" -ldflags "-s -w" .
