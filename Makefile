
version = v0.29.0

goBinary = go
gcflags = -c 3 -B -wb=false
ldflags = -s -w
ldflags_version = $(ldflags) -X 'github.com/jetsetilly/gopher2600/version.number=$(version)'
profilingRom = /home/steve/Desktop/2600_dev/zackattack/waterbed-bouncers-2600/source/bouncers.bin

# the renderer to use for the GUI
#
# the supported renderers are OpenGL 3.2 and OpenGL 2.1
#
# to target OpenGL 2.1 set the renderer variable to gl21
# any other value will target OpenGL 3.2
ifndef renderer
	renderer = gl32
endif


### support targets
.PHONY: all clean tidy generate check_git check_glsl glsl_validate check_pandoc readme_spell

all:
	@echo "use release target to build release binary"

clean:
	@echo "removing binary, profiling files and windows manifests"
	@rm -f gopher2600_* gopher2600
	@rm -f *.profile
	@rm -f rsrc_windows_amd64.syso
	@find ./ -type f | grep "\.orig" | xargs -r rm

tidy:
# goimports is not part of the standard Go distribution so we won't won't
# require this in any of the other targets
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


### testing targets
.PHONY: test race race_debug

test:
# testing with shuffle preferred but it's only available in go 1.17 onwards
ifeq ($(shell $(goBinary) version | awk '{print($$3 >= "go1.17.0")}'), 1)
	$(goBinary) test -shuffle on ./...
else
	$(goBinary) test ./...
endif

race: generate test
	$(goBinary) run -race gopher2600.go $(profilingRom)

race_debug: generate test
	$(goBinary) run -race gopher2600.go debug $(profilingRom)


### profiling targets
.PHONY: profile profile_cpu profile_cpu_play profile_cpu_debug profile_mem_play profile_mem_debug profile_trace

ldflags_profile= $(ldflags) -X 'github.com/jetsetilly/gopher2600/version.number=profiling'

profile:
	@echo use make targets profile_cpu, profile_mem, etc.

profile_cpu: generate test
	@$(goBinary) build -gcflags "$(gcflags)" -ldflags "$(ldflags_profile)"
	@echo "performance mode running for 20s"
	@./gopher2600 performance -profile=cpu -duration=20s $(profilingRom)
	@$(goBinary) tool pprof -http : ./gopher2600 performance_cpu.profile

profile_cpu_play: generate test
	@$(goBinary) build -gcflags "$(gcflags)" -ldflags "$(ldflags_profile)"
	@echo "use window close button to end (CTRL-C will quit the Makefile script)"
	@./gopher2600 play -profile=cpu -elf=none $(profilingRom)
	@$(goBinary) tool pprof -http : ./gopher2600 play_cpu.profile

profile_cpu_debug : generate test
	@$(goBinary) build -gcflags "$(gcflags)" -ldflags "$(ldflags_profile)"
	@echo "use window close button to end (CTRL-C will quit the Makefile script)"
	@./gopher2600 debug -profile=cpu -elf=none $(profilingRom)
	@$(goBinary) tool pprof -http : ./gopher2600 debugger_cpu.profile

profile_mem_play : generate test
	@$(goBinary) build -gcflags "$(gcflags)" -ldflags "$(ldflags_profile)"
	@echo "use window close button to end (CTRL-C will quit the Makefile script)"
	@./gopher2600 play -profile=mem -elf=none $(profilingRom)
	@$(goBinary) tool pprof -http : ./gopher2600 play_mem.profile

profile_mem_debug : generate test
	@$(goBinary) build -gcflags "$(gcflags)" -ldflags "$(ldflags_profile)"
	@echo "use window close button to end (CTRL-C will quit the Makefile script)"
	@./gopher2600 debug -profile=mem -elf=none $(profilingRom)
	@$(goBinary) tool pprof -http : ./gopher2600 debugger_mem.profile

profile_trace: generate test
	@$(goBinary) build -gcflags "$(gcflags)" -ldflags "$(ldflags_profile)"
	@echo "performance mode running for 20s"
	@./gopher2600 performance -profile=trace -duration=20s $(profilingRom)
	@$(goBinary) tool trace -http : performance_trace.profile


### binary building targets for host platform
.PHONY: fontrendering build 

# whether or not we build with freetype font rendering depends on the platform.
# for now freetype doesn't seem to work on MacOS with an M2 CPU
# (it might be better to use the go env variable GOOS AND GOARCH for this)
fontrendering:
ifneq ($(shell $(goBinary) version | grep -q "darwin/arm64"; echo $$?), 0)
# as of v0.29.0 freetype font rendering is disabled for all targets
#fontrendering=imguifreetype
endif

build: fontrendering generate 
	$(goBinary) build -pgo=auto -gcflags "$(gcflags)" -trimpath -ldflags "$(ldflags_version)" -tags="$(fontrendering) $(renderer)"

### release building

.PHONY: version_check release

version_check :
ifndef version
	$(error version is undefined)
endif

release: version_check fontrendering generate 
	$(goBinary) build -pgo=auto -gcflags "$(gcflags)" -trimpath -ldflags "$(ldflags_version)" -tags="$(fontrendering) $(renderer) release"
	mv gopher2600 gopher2600_$(shell go env GOHOSTOS)_$(shell go env GOHOSTARCH)


### cross compilation for windows (tested when cross compiling from Linux)
.PHONY: check_rscr windows_manifest cross_windows_release cross_windows_development cross_winconsole_development

check_rscr:
ifeq (, $(shell which rsrc))
	$(error "rsrc not installed. https://github.com/akavel/rsrc")
endif

windows_manifest: check_rscr
	rsrc -ico .resources/256x256.ico,.resources/48x48.ico,.resources/32x32.ico,.resources/16x16.ico

cross_windows_release: version_check windows_manifest generate
	CGO_ENABLED="1" CC="/usr/bin/x86_64-w64-mingw32-gcc" CXX="/usr/bin/x86_64-w64-mingw32-g++" GOOS="windows" GOARCH="amd64" CGO_LDFLAGS="-static -static-libgcc -static-libstdc++ -L/usr/local/x86_64-w64-mingw32/lib" $(goBinary) build -pgo=auto -tags "static imguifreetype release" -gcflags "$(gcflags)" -trimpath -ldflags "$(ldflags_version) -H=windowsgui" -o gopher2600_windows_amd64.exe .
	rm rsrc_windows_amd64.syso

# intentionally using ldflags and not ldflags for
# cross_windows_development and cross_winconsole_development

cross_windows_development: windows_manifest generate
	CGO_ENABLED="1" CC="/usr/bin/x86_64-w64-mingw32-gcc" CXX="/usr/bin/x86_64-w64-mingw32-g++" GOOS="windows" GOARCH="amd64" CGO_LDFLAGS="-static -static-libgcc -static-libstdc++ -L/usr/local/x86_64-w64-mingw32/lib" $(goBinary) build -pgo=auto -tags "static imguifreetype release" -gcflags "$(gcflags)" -trimpath -ldflags "$(ldflags_version) -H=windowsgui" -o gopher2600_windows_amd64_$(shell git rev-parse --short HEAD).exe .
	rm rsrc_windows_amd64.syso

cross_winconsole_development: windows_manifest generate
	CGO_ENABLED="1" CC="/usr/bin/x86_64-w64-mingw32-gcc" CXX="/usr/bin/x86_64-w64-mingw32-g++" GOOS="windows" GOARCH="amd64" CGO_LDFLAGS="-static -static-libgcc -static-libstdc++ -L/usr/local/x86_64-w64-mingw32/lib" $(goBinary) build -pgo=auto -tags "static imguifreetype release" -gcflags "$(gcflags)" -trimpath -ldflags "$(ldflags_version) -H=windows" -o gopher2600_winconsole_amd64_$(shell git rev-parse --short HEAD).exe .
	rm rsrc_windows_amd64.syso
