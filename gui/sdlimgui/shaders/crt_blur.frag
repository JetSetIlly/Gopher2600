#version 150

uniform sampler2D Texture;
uniform vec2 Blur;
in vec2 Frag_UV;
in vec4 Frag_Color;
out vec4 Out_Color;

void main()
{
	vec4 sum = texture(Texture, Frag_UV) * 0.2270270270;
	sum += texture(Texture, vec2( Frag_UV.x - 4.0 * Blur.x, Frag_UV.y - 4.0 * Blur.y ) ) * 0.0162162162;
	sum += texture(Texture, vec2( Frag_UV.x - 3.0 * Blur.x, Frag_UV.y - 3.0 * Blur.y ) ) * 0.0540540541;
	sum += texture(Texture, vec2( Frag_UV.x - 2.0 * Blur.x, Frag_UV.y - 2.0 * Blur.y ) ) * 0.1216216216;
	sum += texture(Texture, vec2( Frag_UV.x - 1.0 * Blur.x, Frag_UV.y - 1.0 * Blur.y ) ) * 0.1945945946;
	sum += texture(Texture, vec2( Frag_UV.x + 1.0 * Blur.x, Frag_UV.y + 1.0 * Blur.y ) ) * 0.1945945946;
	sum += texture(Texture, vec2( Frag_UV.x + 2.0 * Blur.x, Frag_UV.y + 2.0 * Blur.y ) ) * 0.1216216216;
	sum += texture(Texture, vec2( Frag_UV.x + 3.0 * Blur.x, Frag_UV.y + 3.0 * Blur.y ) ) * 0.0540540541;
	sum += texture(Texture, vec2( Frag_UV.x + 4.0 * Blur.x, Frag_UV.y + 4.0 * Blur.y ) ) * 0.0162162162;
	Out_Color = sum;
}
