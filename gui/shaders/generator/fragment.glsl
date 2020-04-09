// bending and colour splitting in fragment shader cribbed from shadertoy
// project: https://www.shadertoy.com/view/4sf3Dr

uniform int ImageType;
uniform int PixelPerfect;
uniform vec2 Resolution;
uniform sampler2D Texture;
in vec2 Frag_UV;
in vec4 Frag_Color;
out vec4 Out_Color;


void main()
{
	// imgui textures
	if (ImageType != 1) {
		Out_Color = vec4(Frag_Color.rgb, Frag_Color.a * texture(Texture, Frag_UV.st).r);
		return;
	}

	// tv screen texture
	if (PixelPerfect == 1) {
		Out_Color = Frag_Color * texture(Texture, Frag_UV.st);
		return;
	}

	vec2 coords = Frag_UV;

	// split color channels
	vec2 split;
	split.x = 0.001;
	split.y = 0.001;
	Out_Color.r = texture(Texture, vec2(coords.x-split.x, coords.y)).r;
	Out_Color.g = texture(Texture, vec2(coords.x, coords.y+split.y)).g;
	Out_Color.b = texture(Texture, vec2(coords.x+split.x, coords.y)).b;
	Out_Color.a = Frag_Color.a;

	// vignette effect
	float vignette;
	vignette = (10.0*coords.x*coords.y*(1.0-coords.x)*(1.0-coords.y));
	Out_Color.r *= pow(vignette, 0.15) * 1.4;
	Out_Color.g *= pow(vignette, 0.2) * 1.3;
	Out_Color.b *= pow(vignette, 0.2) * 1.3;

	// scanline effect
	if (mod(floor(gl_FragCoord.y), 3.0) == 0.0) {
		Out_Color.a = Frag_Color.a * 0.75;
	}

	// bend screen
	/* float xbend = 6.0; */
	/* float ybend = 5.0; */
	/* coords = (coords - 0.5) * 1.85; */
	/* coords *= 1.11; */	
	/* coords.x *= 1.0 + pow((abs(coords.y) / xbend), 2.0); */
	/* coords.y *= 1.0 + pow((abs(coords.x) / ybend), 2.0); */
	/* coords  = (coords / 2.05) + 0.5; */

	/* // crop tiling */
	/* if (coords.x < 0.0 || coords.x > 1.0 || coords.y < 0.0 || coords.y > 1.0 ) { */
	/* 	discard; */
	/* } */
}
