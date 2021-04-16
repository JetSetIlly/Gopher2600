#version 150

uniform sampler2D Texture;
in vec2 Frag_UV;
in vec4 Frag_Color;
out vec4 Out_Color;

uniform float Alpha;

void main()
{
	Out_Color = Frag_Color * texture(Texture, Frag_UV.st);
	Out_Color.a *= Alpha;
}
