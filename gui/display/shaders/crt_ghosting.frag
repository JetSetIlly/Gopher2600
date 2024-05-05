uniform sampler2D Texture;
uniform vec2 ScreenDim;
uniform float Amount;
in vec2 Frag_UV;
in vec4 Frag_Color;
out vec4 Out_Color;

void main()
{
	// scale amount value by a fixed amount. scaling by some factor of
	// ScreenDim was tried but it proved unsatisfactory
	float amount = Amount/1200;

    vec4 l = texture(Texture, Frag_UV);
    vec4 r = texture(Texture, Frag_UV - vec2(amount, 0.0));
    vec2 f = fract(Frag_UV * ScreenDim);
    Out_Color = mix(l, r, f.x);
}
