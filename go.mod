module gopher2600

go 1.13

require (
	github.com/go-audio/audio v1.0.0
	github.com/go-audio/wav v1.0.0
	github.com/go-gl/gl v0.0.0-20190320180904-bf2b1f2f34d7
	github.com/inkyblackness/imgui-go/v2 v2.1.2-0.20200222162349-d2960522c721
	github.com/pkg/term v0.0.0-20190109203006-aa71e9d9e942
	github.com/veandco/go-sdl2 v0.3.3
	golang.org/x/sys v0.0.0-20191206220618-eeba5f6aabab // indirect
)

replace github.com/inkyblackness/imgui-go/v2 v2.1.2-0.20200222162349-d2960522c721 => github.com/JetSetIlly/imgui-go/v2 v2.1.2-0.20200305084621-2adc4a4bed4e

// replace github.com/inkyblackness/imgui-go/v2 v2.1.2-0.20200222162349-d2960522c721 => ../imgui-go
