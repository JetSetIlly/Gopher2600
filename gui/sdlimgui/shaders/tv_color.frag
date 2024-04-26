uniform sampler2D Texture;
uniform float Brightness;
uniform float Contrast;
uniform float Saturation;
uniform float Hue;
in vec2 Frag_UV;
in vec4 Frag_Color;
out vec4 Out_Color;

#define PI 3.14159

void main()
{
	Out_Color = Frag_Color * texture(Texture, Frag_UV);

	// RGB to YIQ conversions taken from:
	// https://en.wikipedia.org/w/index.php?title=YIQ&oldid=1220238306

	// RGB to YIQ
	mat3 adjust = mat3(
		vec3(0.299, 0.587, 0.114),
		vec3(0.5959, -0.2746, -0.3213),
		vec3(0.2115, -0.5227, 0.3112)
	);

	// contrast. black level and white level is calculated as 10% of the
	// contrast value
	float contrast = Contrast;
	float whiteLevel = 1.0-(contrast*0.1);
    float blackLevel = contrast*0.1;
	float videoLevel = whiteLevel - blackLevel;
    if (videoLevel > 0) {
	    contrast /= videoLevel;
	} else {
        contrast = 0;
	}
	if (contrast < 0) {
		contrast = 0;
	}
	adjust *= contrast;

	// hue
	float hue = 2 * PI * Hue;
	adjust *= mat3(
		vec3(1, 0, 0),
		vec3(0, cos(hue), -sin(hue)),
		vec3(0, sin(hue), cos(hue))
	);

	// YIQ to RGB
	adjust *= mat3(
		vec3(1, 0.956, 0.619),
		vec3(1, -0.272, -0.647),
		vec3(1, -1.106, 1.703)
	);

	// Brightness
	adjust += Brightness - 1.0;

	// apply adjustment
	Out_Color.rgb *= adjust;
}
