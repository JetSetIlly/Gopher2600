#version 150

uniform sampler2D Texture;
uniform sampler2D TextureB;
uniform float Modulate;
in vec2 Frag_UV;
in vec4 Frag_Color;
out vec4 Out_Color;

void main()
{
	vec4 a = texture(Texture, Frag_UV) * vec4(Modulate);
	vec4 b = texture(TextureB, Frag_UV);
	Out_Color = max(a, b * 0.96);
}
