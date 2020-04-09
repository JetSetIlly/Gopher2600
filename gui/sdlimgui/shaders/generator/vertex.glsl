uniform int ImageType;
uniform int PixelPerfect;
uniform vec2 Resolution;
uniform mat4 ProjMtx;
in vec2 Position;
in vec2 UV;
in vec4 Color;
out vec2 Frag_UV;
out vec4 Frag_Color;

void main()
{
	// imgui textures
	if (ImageType != 1) {
		Frag_UV = UV;
		Frag_Color = Color;
		gl_Position = ProjMtx * vec4(Position.xy,0,1);
		return;
	}

	// tv screen texture
	Frag_UV = UV;
	Frag_Color = Color;
	gl_Position = ProjMtx * vec4(Position.xy,0,1);
}
