#version 150

uniform sampler2D Texture;
uniform sampler2D NewFrame;
uniform float Latency;
uniform int CorrectVideoBlack; // 1 == true; 0 == false
in vec2 Frag_UV;
in vec4 Frag_Color;
out vec4 Out_Color;

void main()
{
	vec4 a = texture(Texture, Frag_UV) * vec4(Latency);
	vec4 b = texture(NewFrame, Frag_UV);
	Out_Color = max(a, b * 0.96);

	// video-black correction
	if (CorrectVideoBlack == 1) {
		float vb = 0.06;
		Out_Color.rgb = clamp(Out_Color.rgb, vec3(vb), Out_Color.rgb);
	}
}
