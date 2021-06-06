#version 150

uniform sampler2D Texture;
uniform sampler2D NewFrame;
in vec2 Frag_UV;
in vec4 Frag_Color;
out vec4 Out_Color;

void main()
{
	float vb = 0.05;
	Out_Color = texture(Texture, Frag_UV);
	Out_Color.rgb = clamp(Out_Color.rgb, vec3(vb), Out_Color.rgb);
}
