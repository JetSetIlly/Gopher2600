compileFlags = '-c 3 -B -wb=false'

# profilingRom = roms/Homebrew/CDF/galaga_dmo_v2_NTSC.bin
# profilingRom = roms/Homebrew/DPC+ARM/ZaxxonHDDemo_150927_NTSC.bin
# profilingRom = roms/Rsboxing.bin
# profilingRom = "test_roms/plusrom/sokoboo Plus.bin"
# profilingRom = "roms/starpath/02 - Communist Mutants From Space (Ntsc).mp3"
# profilingRom = roms/Homebrew/CDF/gorfarc_20201231_demo1_NTSC.bin
profilingRom = roms/Pitfall.bin

.PHONY: all clean tidy generate check_lint lint check_glsl glsl_validate check_pandoc readme_spell test race race_debug profile profile_cpu profile_mem profile_trace build_assertions build check_upx release release_statsview cross_windows cross_windows_statsview binaries check_gotip build_with_gotip

goBinary = go
# goBinary = ~/Go/dev_github/go/bin/go
# goBinary = gotip

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
	@$(goBinary) generate ./...

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
	@glslangValidator gui/sdlimgui/shaders/*.vert
	@glslangValidator gui/sdlimgui/shaders/*.frag

check_pandoc:
ifeq (, $(shell which pandoc))
	$(error "pandoc not installed")
endif

readme_spell: check_pandoc
	@pandoc README.md -t plain | aspell -a | cut -d ' ' -f 2 | awk 'length($0)>1' | sort | uniq


test:
# testing with shuffle on is good but it's only available in go 1.17 onwards
ifeq ($(shell $(goBinary) version | grep -q 1.17 ; echo $$?), 0)
	$(goBinary) test -shuffle on ./...
else
	$(goBinary) test ./...
endif

race: generate test
	$(goBinary) run -race gopher2600.go $(profilingRom)

race_debug: generate test
	$(goBinary) run -race gopher2600.go debug $(profilingRom)

profile:
	@echo use make targets profile_cpu, profile_mem or profile_trace

profile_cpu: generate test
	@$(goBinary) build -gcflags $(compileFlags)
	@echo "performance mode running for 20s"
	@./gopher2600 performance --profile=cpu --fpscap=false --duration=20s $(profilingRom)
	@$(goBinary) tool pprof -http : ./gopher2600 performance_cpu.profile

profile_cpu_display: generate test
	@$(goBinary) build -gcflags $(compileFlags)
	@echo "performance mode running for 20s"
	@./gopher2600 performance --profile=cpu --fpscap=false --display --duration=20s $(profilingRom)
	@$(goBinary) tool pprof -http : ./gopher2600 performance_cpu.profile

profile_cpu_again:
	@$(goBinary) tool pprof -http : ./gopher2600 performance_cpu.profile

profile_mem: generate test
	@$(goBinary) build -gcflags $(compileFlags)
	@echo "use window close button to end (CTRL-C will quit the Makefile script)"
	@./gopher2600 debug --profile=mem $(profilingRom)
	@$(goBinary) tool pprof -http : ./gopher2600 debugger_mem.profile

profile_trace: generate test
	@$(goBinary) build -gcflags $(compileFlags)
	@echo "performance mode running for 20s"
	@./gopher2600 performance --profile=trace --fpscap=false --duration=20s $(profilingRom)
	@$(goBinary) tool trace -http : performance_trace.profile

build_assertions: generate test
	$(goBinary) build -gcflags $(compileFlags) -tags=assertions

# deliberately not having test dependecies for remaining targets

build: generate 
	$(goBinary) build -gcflags $(compileFlags)

build_statsview: generate 
	$(goBinary) build -gcflags $(compileFlags) -tags="statsview" -o gopher2600_statsview

check_upx:
ifeq (, $(shell which upx))
	$(error "upx not installed")
endif

release: check_upx generate 
	$(goBinary) build -gcflags $(compileFlags) -ldflags="-s -w" -tags="release"
	upx -o gopher2600.upx gopher2600
	mv gopher2600.upx gopher2600_$(shell go env GOHOSTOS)_$(shell go env GOHOSTARCH)
	rm gopher2600

release_statsview: check_upx generate 
	$(goBinary) build -gcflags $(compileFlags) -ldflags="-s -w" -tags="release statsview" -o gopher2600_statsview
	upx -o gopher2600_statsview.upx gopher2600_statsview
	mv gopher2600_statsview.upx gopher2600_statsview_$(shell go env GOHOSTOS)_$(shell go env GOHOSTARCH)
	rm gopher2600_statsview

# cross_windows_dynamic: generate 
# 	CGO_ENABLED="1" CC="/usr/bin/x86_64-w64-mingw32-gcc" CXX="/usr/bin/x86_64-w64-mingw32-g++" GOOS="windows" GOARCH="amd64" CGO_LDFLAGS="-lmingw32 -lSDL2" CGO_CFLAGS="-D_REENTRANT" go build -tags "release" -gcflags $(compileFlags) -ldflags="-s -w" .

cross_windows: generate 
	CGO_ENABLED="1" CC="/usr/bin/x86_64-w64-mingw32-gcc" CXX="/usr/bin/x86_64-w64-mingw32-g++" GOOS="windows" GOARCH="amd64" CGO_LDFLAGS="-static-libgcc -static-libstdc++" $(goBinary) build -tags "static release" -gcflags $(compileFlags) -ldflags "-s -w" -o gopher2600_windows_amd64.exe .

cross_windows_statsview: generate 
	CGO_ENABLED="1" CC="/usr/bin/x86_64-w64-mingw32-gcc" CXX="/usr/bin/x86_64-w64-mingw32-g++" GOOS="windows" GOARCH="amd64" CGO_LDFLAGS="-static-libgcc -static-libstdc++" $(goBinary) build -tags "static release statsview" -gcflags $(compileFlags) -ldflags "-s -w" -o gopher2600_statsview_windows_amd64.exe .

binaries: release release_statsview cross_windows cross_windows_statsview
	@echo "build release binaries"
