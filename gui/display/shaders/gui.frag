uniform sampler2D Texture;
in vec2 Frag_UV;
in vec4 Frag_Color;
out vec4 Out_Color;

void main()
{
	Out_Color = vec4(Frag_Color.rgb, Frag_Color.a * texture(Texture, Frag_UV).r);
}
