gcflags = '-c 3 -B -wb=false'

# profilingRom = roms/Rsboxing.bin
# profilingRom = "roms/starpath/02 - Communist Mutants From Space (Ntsc).mp3"
# profilingRom = "roms/The Official Frogger.bin"
# profilingRom = roms/Homebrew/CDF/zookeeper_20200308_demo2_NTSC.bin
# profilingRom = roms/Pitfall.bin
# profilingRom = test_roms/ELF/raycaster/raycaster.bin
# profilingRom = /home/steve/Desktop/2600_dev/davie/Boulder-Dash-CDFJ-NG/CDFJBoulderDash.bin
profilingRom = /home/steve/Desktop/2600_dev/marcoj/RPG/build/RPG_4K_60HZ.ace
# profilingRom = test_roms/ACE/4k/cartridge/cartridge.ace

.PHONY: all clean tidy generate check_glsl glsl_validate check_pandoc readme_spell test race race_debug profile profile_cpu profile_cpu_play profile_cpu_debug profile_mem_play profile_mem_debug profile_trace build_assertions build release windows_manifest cross_windows cross_windows_development cross_winconsole_development cross_windows_dynamic

goBinary = go

all:
	@echo "use release target to build release binary"

clean:
	@echo "removing binary, profiling files and windows manifests"
	@rm -f gopher2600_* gopher2600
	@rm -f *.profile
	@rm -f rsrc_windows_amd64.syso
	@find ./ -type f | grep "\.orig" | xargs -r rm

tidy:
	goimports -w .

generate:
	@$(goBinary) generate ./...

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
	@pandoc README.md -t plain | aspell -a | sed '1d;$d' | cut -d ' ' -f 2 | awk 'length($0)>1' | sort | uniq
# sed is used to chop off the first line of aspell output, which is a version banner

test:
# testing with shuffle on is good but it's only available in go 1.17 onwards
ifeq ($(shell $(goBinary) version | awk '{print($$3 >= "go1.17.0")}'), 1)
	$(goBinary) test -shuffle on ./...
else
	$(goBinary) test ./...
endif

race: generate test
	$(goBinary) run -race gopher2600.go $(profilingRom)

race_debug: generate test
	$(goBinary) run -race gopher2600.go debug $(profilingRom)

profile:
	@echo use make targets profile_cpu, profile_mem, etc.

profile_cpu: generate test
	@$(goBinary) build -gcflags $(gcflags)
	@echo "performance mode running for 20s"
	@./gopher2600 performance --profile=cpu --duration=20s $(profilingRom)
	@$(goBinary) tool pprof -http : ./gopher2600 performance_cpu.profile

profile_cpu_play: generate test
	@$(goBinary) build -gcflags $(gcflags)
	@echo "use window close button to end (CTRL-C will quit the Makefile script)"
	@./gopher2600 play --profile=cpu -elf none $(profilingRom)
	@$(goBinary) tool pprof -http : ./gopher2600 play_cpu.profile

profile_cpu_debug : generate test
	@$(goBinary) build -gcflags $(gcflags)
	@echo "use window close button to end (CTRL-C will quit the Makefile script)"
	@./gopher2600 debug --profile=cpu $(profilingRom)
	@$(goBinary) tool pprof -http : ./gopher2600 debugger_cpu.profile

profile_mem_play : generate test
	@$(goBinary) build -gcflags $(gcflags)
	@echo "use window close button to end (CTRL-C will quit the Makefile script)"
	@./gopher2600 play --profile=mem $(profilingRom)
	@$(goBinary) tool pprof -http : ./gopher2600 play_mem.profile

profile_mem_debug : generate test
	@$(goBinary) build -gcflags $(gcflags)
	@echo "use window close button to end (CTRL-C will quit the Makefile script)"
	@./gopher2600 debug --profile=mem $(profilingRom)
	@$(goBinary) tool pprof -http : ./gopher2600 debugger_mem.profile

profile_trace: generate test
	@$(goBinary) build -gcflags $(gcflags)
	@echo "performance mode running for 20s"
	@./gopher2600 performance --profile=trace --duration=20s $(profilingRom)
	@$(goBinary) tool trace -http : performance_trace.profile

build_assertions: generate test
	$(goBinary) build -gcflags $(gcflags) -tags="assertions"

# deliberately not having test dependecies for remaining targets

# whether or not we build with freetype font rendering depends on the platform.
# for now freetype doesn't seem to work on MacOS with an M2 CPU
#
# we should really use the go env variable GOOS AND GOARCH for this
fontrendering:
ifneq ($(shell $(goBinary) version | grep -q "darwin/arm64"; echo $$?), 0)
fontrendering=imguifreetype
endif

build: fontrendering generate 
	$(goBinary) build -pgo=auto -gcflags $(gcflags) -trimpath -tags="$(fontrendering)"

release: fontrendering generate 
	$(goBinary) build -pgo=auto -gcflags $(gcflags) -trimpath -ldflags="-s -w" -tags="$(fontrendering) release"
	mv gopher2600 gopher2600_$(shell go env GOHOSTOS)_$(shell go env GOHOSTARCH)

## windows cross compilation (tested when cross compiling from Linux)

check_rscr:
ifeq (, $(shell which rsrc))
	$(error "rsrc not installed. https://github.com/akavel/rsrc")
endif

windows_manifest: check_rscr
	rsrc -ico .resources/256x256.ico,.resources/48x48.ico,.resources/32x32.ico,.resources/16x16.ico

cross_windows: generate windows_manifest
	CGO_ENABLED="1" CC="/usr/bin/x86_64-w64-mingw32-gcc" CXX="/usr/bin/x86_64-w64-mingw32-g++" GOOS="windows" GOARCH="amd64" CGO_LDFLAGS="-static -static-libgcc -static-libstdc++ -L/usr/local/x86_64-w64-mingw32/lib" $(goBinary) build -pgo=auto -tags "static imguifreetype release" -gcflags $(gcflags) -trimpath -ldflags "-s -w -H=windowsgui" -o gopher2600_windows_amd64.exe .
	rm rsrc_windows_amd64.syso

cross_windows_development: generate windows_manifest
	CGO_ENABLED="1" CC="/usr/bin/x86_64-w64-mingw32-gcc" CXX="/usr/bin/x86_64-w64-mingw32-g++" GOOS="windows" GOARCH="amd64" CGO_LDFLAGS="-static -static-libgcc -static-libstdc++ -L/usr/local/x86_64-w64-mingw32/lib" $(goBinary) build -pgo=auto -tags "static imguifreetype release" -gcflags $(gcflags) -trimpath -ldflags "-s -w -H=windowsgui" -o gopher2600_windows_amd64_$(shell git rev-parse --short HEAD).exe .
	rm rsrc_windows_amd64.syso

cross_winconsole_development: generate windows_manifest
	CGO_ENABLED="1" CC="/usr/bin/x86_64-w64-mingw32-gcc" CXX="/usr/bin/x86_64-w64-mingw32-g++" GOOS="windows" GOARCH="amd64" CGO_LDFLAGS="-static -static-libgcc -static-libstdc++ -L/usr/local/x86_64-w64-mingw32/lib" $(goBinary) build -pgo=auto -tags "static imguifreetype release" -gcflags $(gcflags) -trimpath -ldflags "-s -w -H=windows" -o gopher2600_winconsole_amd64_$(shell git rev-parse --short HEAD).exe .
	rm rsrc_windows_amd64.syso

# cross_windows_dynamic: generate windows_manifest
# 	CGO_ENABLED="1" CC="/usr/bin/x86_64-w64-mingw32-gcc" CXX="/usr/bin/x86_64-w64-mingw32-g++" GOOS="windows" GOARCH="amd64" CGO_LDFLAGS="-lmingw32 -lSDL2" CGO_CFLAGS="-D_REENTRANT" go build -pgo=auto -tags "release" -gcflags $(gcflags) -trimpath -ldflags="-s -w -H=windowsgui" -o gopher2600_windows_amd64.exe .
