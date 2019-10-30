all:
	@echo "use release target to build release binary"

generate:
	@go generate ./...

clean:
	@echo "removing binary and profiling files"
	@rm -f gopher2600 cpu.profile mem.profile

release:
	go build -gcflags '-c 3 -B -+ -wb=false' .

profile:
	go build -gcflags '-c 3 -B -+ -wb=false' .
	./gopher2600 performance --profile roms/ROMs/Pitfall.bin
	go tool pprof -http : ./gopher2600 cpu.profile

profile_display:
	go build -gcflags '-c 3 -B -+ -wb=false' .
	./gopher2600 performance --display --profile roms/ROMs/Pitfall.bin
	go tool pprof -http : ./gopher2600 cpu.profile
