uniform sampler2D Texture;
uniform float Time;
in vec2 Frag_UV;
in vec4 Frag_Color;
out vec4 Out_Color;

void main()
{
	Out_Color = Frag_Color * texture(Texture, Frag_UV);

	// add a small amount of perlin noise to the image
	if (Out_Color.a >= 1.0) {
		float perlin = fract(sin(dot(Frag_UV, vec2(12.9898, 78.233))*Time) * 43758.5453);
		perlin *= 0.03;
		Out_Color.rgb += vec3(perlin);
	}
}
