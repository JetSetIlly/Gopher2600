uniform sampler2D Texture;
uniform sampler2D NewFrame;
uniform float Latency;
in vec2 Frag_UV;
in vec4 Frag_Color;
out vec4 Out_Color;

void main()
{
	vec4 a = texture(Texture, Frag_UV) * vec4(Latency);
	vec4 b = texture(NewFrame, Frag_UV);
	Out_Color = max(a, b);
}
