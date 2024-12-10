uniform sampler2D Texture;
uniform int FromGUI;
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
	if (FromGUI == 1) {
		// shader has been run instead of the GUI shader. this means we have to
		// extract the rgba values from the texture slightly differently
		Out_Color = vec4(Frag_Color.rgb, Frag_Color.a * texture(Texture, Frag_UV).r);
	} else {
		Out_Color = Frag_Color * texture(Texture, Frag_UV);
	}

	// RGB to YIQ conversions taken from:
	// https://en.wikipedia.org/w/index.php?title=YIQ&oldid=1220238306

	// RGB to YIQ
	mat3 adjust = mat3(
		vec3(0.299, 0.587, 0.114),
		vec3(0.5959, -0.2746, -0.3213),
		vec3(0.2115, -0.5227, 0.3112)
	);

	// contrast
	float contrast = clamp(Contrast, 0.0, 2.0);
	adjust *= mat3(
		vec3(contrast, 0, 0),
		vec3(0, 1, 0),
		vec3(0, 0, 1)
	);

	// saturation
	float saturation = clamp(Saturation, 0.0, 2.0);
	adjust *= mat3(
		vec3(1, 0, 0),
		vec3(0, Saturation, 0),
		vec3(0, 0, Saturation)
	);

	// hue
	float hue = 2 * PI * Hue;
	adjust *= mat3(
		vec3(1, 0, 0),
		vec3(0, cos(hue), -sin(hue)),
		vec3(0, sin(hue), cos(hue))
	);

	// brightness
	float brightness = clamp(Brightness-1.0, -1.0, 1.0);
	adjust += mat3(
		vec3(brightness, 0, 0),
		vec3(0, 0, 0),
		vec3(0, 0, 0)
	);

	// YIQ to RGB
	adjust *= mat3(
		vec3(1, 0.956, 0.619),
		vec3(1, -0.272, -0.647),
		vec3(1, -1.106, 1.703)
	);

	// apply adjustment
	Out_Color.rgb *= adjust;

	// gamma correct signal for the monitor
	Out_Color.r = pow(Out_Color.r, 2.2);
	Out_Color.g = pow(Out_Color.g, 2.2);
	Out_Color.b = pow(Out_Color.b, 2.2);
}
