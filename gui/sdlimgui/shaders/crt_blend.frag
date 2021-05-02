#version 150

uniform sampler2D Texture;
uniform sampler2D PrevBlend;
uniform float Modulate;
uniform float Fade;
in vec2 Frag_UV;
in vec4 Frag_Color;
out vec4 Out_Color;

void main()
{
	vec4 a = texture(Texture, Frag_UV) * vec4(Modulate);
	vec4 b = texture(PrevBlend, Frag_UV);
	Out_Color = max(a, b * Fade);
}
