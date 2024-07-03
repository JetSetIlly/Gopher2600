uniform sampler2D Texture;
uniform int Stripe;
uniform float Stripe_Size;
uniform float Stripe_Fade;
in vec2 Frag_UV;
in vec4 Frag_Color;
out vec4 Out_Color;

void main()
{
	Out_Color = Frag_Color * texture(Texture, Frag_UV);
	Out_Color = paintingEffect(Frag_UV, Out_Color);

	if (Out_Color.a > 0.0 && Stripe != 0) {
		float x = gl_FragCoord.x / Stripe_Size;
		float y = gl_FragCoord.y / Stripe_Size;
		float sum = x + y;
		if (int(mod(float(sum), float(2))) != 0) {
			Out_Color.rgb *= Stripe_Fade;
		}
	}
}
