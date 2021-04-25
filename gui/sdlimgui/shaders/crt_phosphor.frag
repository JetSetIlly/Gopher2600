#version 150

uniform sampler2D Texture;
uniform sampler2D PrevPhosphor;
uniform float Latency;
in vec2 Frag_UV;
in vec4 Frag_Color;
out vec4 Out_Color;

void main()
{
	vec4 a = texture(Texture, Frag_UV) * vec4(Latency);
	vec4 b = texture(PrevPhosphor, Frag_UV);
	Out_Color = max(a, b * 0.96);
}
