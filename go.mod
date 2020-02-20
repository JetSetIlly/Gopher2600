module gopher2600

go 1.13

require (
	github.com/go-audio/audio v1.0.0
	github.com/go-audio/wav v1.0.0
	github.com/go-gl/gl v0.0.0-20190320180904-bf2b1f2f34d7
	github.com/inkyblackness/imgui-go/v2 v2.1.2-0.20200210203827-9487e0b50076
	github.com/pkg/term v0.0.0-20190109203006-aa71e9d9e942
	github.com/veandco/go-sdl2 v0.3.3
	golang.org/x/sys v0.0.0-20191206220618-eeba5f6aabab // indirect
)

// replace github.com/inkyblackness/imgui-go/v2 v2.1.2-0.20200210203827-9487e0b50076 => github.com/JetSetIlly/imgui-go/v2 v2.1.2-0.20200219162743-bda35fa2a772

replace github.com/inkyblackness/imgui-go/v2 v2.1.2-0.20200210203827-9487e0b50076 => ../imgui-go
