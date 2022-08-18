#version 150

uniform sampler2D Texture;
uniform sampler2D NewFrame;
uniform float BlackLevel;
in vec2 Frag_UV;
in vec4 Frag_Color;
out vec4 Out_Color;

void main()
{
	Out_Color = texture(Texture, Frag_UV);

	// scale black level by +5% (done so that existing default values in
	// people's preferences will be increased)
	Out_Color.rgb = clamp(Out_Color.rgb, vec3(BlackLevel*1.05), Out_Color.rgb);
}
