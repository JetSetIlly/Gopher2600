module github.com/jetsetilly/gopher2600

go 1.14

//replace github.com/inkyblackness/imgui-go/v2 v2.2.0 => github.com/JetSetIlly/imgui-go/v2 v2.2.1-0.20200317095507-51a6e45b93a9

//replace	github.com/inkyblackness/imgui-go/v2 v2.2.0 => ../imgui-go

require (
	github.com/go-audio/audio v1.0.0
	github.com/go-audio/wav v1.0.0
	github.com/go-gl/gl v0.0.0-20190320180904-bf2b1f2f34d7
	github.com/inkyblackness/imgui-go/v2 v2.3.0
	github.com/pkg/term v0.0.0-20190109203006-aa71e9d9e942
	github.com/veandco/go-sdl2 v0.4.1
)
