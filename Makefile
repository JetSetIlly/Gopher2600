
version = v0.53.0-preview

goBinary = go
gcflags = -c 3 -B -wb=false -l -l -l -l
ldflags = -s -w
ldflags_version = $(ldflags) -X 'github.com/jetsetilly/gopher2600/version.number=$(version)'

# the rom to use for the race and profiling targets
ifndef rom
	rom = roms/Pitfall.bin
endif

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
.PHONY: all clean tidy generate glsl_validate readme_spell patch_file_integrity lint
.PHONY: check_glsl check_pandoc check_awk check_linters

all:
	@echo "use 'release' to build release binary"

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

check_awk:
ifeq (, $(shell which awk))
	$(error "awk not installed")
endif

readme_spell: check_pandoc check_awk
	@pandoc README.md -t plain | aspell -a | sed '1d;$d' | cut -d ' ' -f 2 | awk 'length($0)>1' | sort | uniq
# sed is used to chop off the first line of aspell output, which is a version banner

patch_file_integrity: *.patch
# check that patch files are still valid
	@echo "patch file integrity"
	@for file in $^; do \
		echo "  $$file"; \
		git apply --check $$file; \
	done
	@echo "patch files are fine"

check_linters:
# https://github.com/polyfloyd/go-errorlint
ifeq (, $(shell which go-errorlint))
	$(error "go-errorlint not installed")
endif
# https://github.com/mdempsky/unconvert
ifeq (, $(shell which unconvert))
	$(error "unconvert not installed")
endif

lint: check_linters
	go vet ./...
	go-errorlint -c 0 -errorf -errorf-multi ./...
	unconvert ./...

### testing targets
.PHONY: test race race_debug fuzz

test:

# testing with shuffle preferred but it's only available in go 1.17 onwards
ifeq ($(shell $(goBinary) version | awk '{print($$3 >= "go1.17.0")}'), 1)
	# running with -count=1 forces the test to be rerun every time and not rely on
	# cached results
	$(goBinary) test -count=1 -shuffle on ./...
else
	$(goBinary) test -count=1 ./...
endif

race: generate test
	$(goBinary) run -race gopher2600.go "$(rom)"

race_debug: generate test
	$(goBinary) run -race gopher2600.go debug "$(rom)"

fuzz: generate test
# fuzz testing cannot work with multiple packages so we must list the ones
# we want to include
	$(goBinary) test -fuzztime=30s -fuzz . ./crunched/

### profiling targets
.PHONY: profile
.PHONY: profile_cpu profile_mem profile_trace
.PHONY:	profile_cpu_play profile_mem_play profile_trace_play

# not including $(ldflags) for profiling. that woulds strip the binary of
# debugging data which would prevent us from digging into the source code
ldflags_profile = -X 'github.com/jetsetilly/gopher2600/version.number=profiling'

profile:
	@echo use make targets profile_cpu, profile_mem, etc.

profile_cpu: generate test
	@$(goBinary) build -gcflags "$(gcflags)" -ldflags "$(ldflags_profile)"
	@echo "performance mode running for 20s"
	@./gopher2600 performance -profile=cpu -duration=20s "$(rom)"
	@$(goBinary) tool pprof -http : ./gopher2600 performance_cpu.profile

profile_mem : generate test
	@$(goBinary) build -gcflags "$(gcflags)" -ldflags "$(ldflags_profile)"
	@echo "performance mode running for 20s"
	@./gopher2600 performance -profile=mem -duration=20s "$(rom)"
	@$(goBinary) tool pprof -http : ./gopher2600 performance_mem.profile

profile_trace: generate test
	@$(goBinary) build -gcflags "$(gcflags)" -ldflags "$(ldflags_profile)"
	@echo "performance mode running for 20s"
	@./gopher2600 performance -profile=trace -duration=20s "$(rom)"
	@$(goBinary) tool trace -http : performance_trace.profile

profile_cpu_play: generate test
	@$(goBinary) build -gcflags "$(gcflags)" -ldflags "$(ldflags_profile)"
	@echo "use window close button to end (CTRL-C will quit the Makefile script)"
	@./gopher2600 play -profile=cpu -dwarf=none "$(rom)"
	@$(goBinary) tool pprof -http : ./gopher2600 play_cpu.profile

profile_mem_play : generate test
	@$(goBinary) build -gcflags "$(gcflags)" -ldflags "$(ldflags_profile)"
	@echo "use window close button to end (CTRL-C will quit the Makefile script)"
	@./gopher2600 play -profile=mem -dwarf=none "$(rom)"
	@$(goBinary) tool pprof -http : ./gopher2600 play_mem.profile

profile_trace_play: generate test
	@$(goBinary) build -gcflags "$(gcflags)" -ldflags "$(ldflags_profile)"
	@echo "use window close button to end (CTRL-C will quit the Makefile script)"
	@./gopher2600 play -profile=trace -dwarf=none "$(rom)"
	@$(goBinary) tool trace -http : play_trace.profile


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
	CGO_ENABLED="1" CC="/usr/bin/x86_64-w64-mingw32-gcc" CXX="/usr/bin/x86_64-w64-mingw32-g++" GOOS="windows" GOARCH="amd64" CGO_LDFLAGS="-static -static-libgcc -static-libstdc++ -L/usr/local/x86_64-w64-mingw32/lib" $(goBinary) build -pgo=auto -tags "$(fontrendering) $(renderer) static release" -gcflags "$(gcflags)" -trimpath -ldflags "$(ldflags_version) -H=windowsgui" -o gopher2600_windows_amd64.exe .
	rm rsrc_windows_amd64.syso

# intentionally using ldflags and not ldflags for
# cross_windows_development and cross_winconsole_development

cross_windows_development: windows_manifest generate
	CGO_ENABLED="1" CC="/usr/bin/x86_64-w64-mingw32-gcc" CXX="/usr/bin/x86_64-w64-mingw32-g++" GOOS="windows" GOARCH="amd64" CGO_LDFLAGS="-static -static-libgcc -static-libstdc++ -L/usr/local/x86_64-w64-mingw32/lib" $(goBinary) build -pgo=auto -tags "$(fontrendering) $(renderer) static release" -gcflags "$(gcflags)" -trimpath -ldflags "$(ldflags_version) -H=windowsgui" -o gopher2600_windows_amd64_$(shell git rev-parse --short HEAD).exe .
	rm rsrc_windows_amd64.syso

cross_winconsole_development: windows_manifest generate
	CGO_ENABLED="1" CC="/usr/bin/x86_64-w64-mingw32-gcc" CXX="/usr/bin/x86_64-w64-mingw32-g++" GOOS="windows" GOARCH="amd64" CGO_LDFLAGS="-static -static-libgcc -static-libstdc++ -L/usr/local/x86_64-w64-mingw32/lib" $(goBinary) build -pgo=auto -tags "$(fontrendering) $(renderer) static release" -gcflags "$(gcflags)" -trimpath -ldflags "$(ldflags_version) -H=windows" -o gopher2600_winconsole_amd64_$(shell git rev-parse --short HEAD).exe .
	rm rsrc_windows_amd64.syso
