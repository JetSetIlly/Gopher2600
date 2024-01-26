uniform sampler2D Texture;
in vec2 Frag_UV;
in vec4 Frag_Color;
out vec4 Out_Color;

uniform float Amount;

void main()
{
	vec2 uv = Frag_UV;
	uv.y = mod(uv.y - Amount, 1.0);
	Out_Color = Frag_Color * texture(Texture, uv.st);
}
