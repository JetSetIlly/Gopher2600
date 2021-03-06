compileFlags = '-c 3 -B -wb=false'
#profilingRom = roms/Homebrew/CDF/galaga_dmo_v2_NTSC.bin
#profilingRom = roms/Homebrew/DPC+ARM/ZaxxonHDDemo_150927_NTSC.bin
profilingRom = roms/Rsboxing.bin
#profilingRom = "test_roms/plusrom/sokoboo Plus.bin"
#profilingRom = "roms/starpath/02 - Communist Mutants From Space (Ntsc).mp3"

.PHONY: all clean tidy generate check_lint lint check_glsl glsl_validate check_pandoc readme_spell test race profile build_assertions build check_upx release release_statsview cross_windows cross_windows_statsview binaries check_gotip build_with_gotip

all:
	@echo "use release target to build release binary"

clean:
	@echo "removing binary and profiling files"
	@rm -f *.profile
	@rm -f gopher2600_* gopher2600
	@find ./ -type f | grep "\.orig" | xargs -r rm

tidy:
	goimports -w .

generate:
	@go generate ./...

check_lint:
ifeq (, $(shell which golangci-lint))
	$(error "golanci-lint not installed")
endif

lint: check_lint
# uses .golangci.yml configuration file
	golangci-lint run --sort-results

lint_fix: check_lint
# uses .golangci.yml configuration file
	golangci-lint run --fix --sort-results

check_glsl:
ifeq (, $(shell which glslangValidator))
	$(error "glslangValidator not installed")
endif

glsl_validate: check_glsl
	@glslangValidator gui/crt/shaders/generator/fragment.frag
	@glslangValidator gui/crt/shaders/generator/vertex.vert

check_pandoc:
ifeq (, $(shell which pandoc))
	$(error "pandoc not installed")
endif

readme_spell: check_pandoc
	@pandoc README.md -t plain | aspell -a | cut -d ' ' -f 2 | awk 'length($0)>1' | sort | uniq

test:
	go test ./...

race: generate test
# disable checkptr because the opengl implementation will trigger it and cause
# a lot of output noise
	go run -race -gcflags=all=-d=checkptr=0 gopher2600.go debug $(profilingRom)

profile: generate test
	go build -gcflags $(compileFlags)
	@echo
	@echo "WHAT NEXT"
	@echo "---------"
	@echo "1.  run gopher2600 in RUN, DEBUG or PERFORMANCE mode"
	@echo "1a. use --profile argument. CPU, MEM, TRACE for profiling types (comma separated)"
	@echo "2.  view cpu and mem profile with: go tool pprof -http : ./gopher2600 <profile>"
	@echo "2b. view trace with: go tool trace -http : <trace_profile>"

build_assertions: generate test
	go build -gcflags $(compileFlags) -tags=assertions

# deliberately not having test dependecies for remaining targets

build: generate 
	go build -gcflags $(compileFlags)

build_statsview: generate 
	go build -gcflags $(compileFlags) -tags="statsview" -o gopher2600_statsview

check_upx:
ifeq (, $(shell which upx))
	$(error "upx not installed")
endif

release: check_upx generate 
	go build -gcflags $(compileFlags) -ldflags="-s -w" -tags="release"
	upx -o gopher2600.upx gopher2600
	mv gopher2600.upx gopher2600_$(shell go env GOHOSTOS)_$(shell go env GOHOSTARCH)
	rm gopher2600

release_statsview: check_upx generate 
	go build -gcflags $(compileFlags) -ldflags="-s -w" -tags="release statsview" -o gopher2600_statsview
	upx -o gopher2600_statsview.upx gopher2600_statsview
	mv gopher2600_statsview.upx gopher2600_statsview_$(shell go env GOHOSTOS)_$(shell go env GOHOSTARCH)
	rm gopher2600_statsview

# cross_windows_dynamic: generate 
# 	CGO_ENABLED="1" CC="/usr/bin/x86_64-w64-mingw32-gcc" CXX="/usr/bin/x86_64-w64-mingw32-g++" GOOS="windows" GOARCH="amd64" CGO_LDFLAGS="-lmingw32 -lSDL2" CGO_CFLAGS="-D_REENTRANT" go build -tags "release" -gcflags $(compileFlags) -ldflags="-s -w" .

cross_windows: generate 
	CGO_ENABLED="1" CC="/usr/bin/x86_64-w64-mingw32-gcc" CXX="/usr/bin/x86_64-w64-mingw32-g++" GOOS="windows" GOARCH="amd64" CGO_LDFLAGS="-static-libgcc -static-libstdc++" go build -tags "static release" -gcflags $(compileFlags) -ldflags "-s -w" -o gopher2600_windows_amd64.exe .

cross_windows_statsview: generate 
	CGO_ENABLED="1" CC="/usr/bin/x86_64-w64-mingw32-gcc" CXX="/usr/bin/x86_64-w64-mingw32-g++" GOOS="windows" GOARCH="amd64" CGO_LDFLAGS="-static-libgcc -static-libstdc++" go build -tags "static release statsview" -gcflags $(compileFlags) -ldflags "-s -w" -o gopher2600_statsview_windows_amd64.exe .

binaries: release release_statsview cross_windows cross_windows_statsview
	@echo "build release binaries"

check_gotip:
ifeq (, $(shell which gotip))
	$(error "gotip not installed")
endif

build_with_gotip: check_gotip generate
	gotip build -gcflags $(compileFlags)
