uniform mat4 ProjMtx;
uniform int FlipY;
in vec2 Position;
in vec2 UV;
in vec4 Color;
out vec2 Frag_UV;
out vec4 Frag_Color;

void main()
{
	Frag_UV = UV;
	Frag_Color = Color;
	gl_Position = ProjMtx * vec4(Position.xy,0,1);
	if (FlipY != 0) {
		gl_Position.y *= -1;
	}
}
