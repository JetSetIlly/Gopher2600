package shaders

import _ "embed"

//go:embed "vertex.vert"
var VertexShader []byte

//go:embed "gui.frag"
var GUIShader []byte

//go:embed "color.frag"
var ColorShader []byte
