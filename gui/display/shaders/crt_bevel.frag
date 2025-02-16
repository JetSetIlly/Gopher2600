uniform sampler2D Texture;
uniform float Time;

uniform int Rim;
uniform sampler2D Screen;

uniform int AmbientTint;
uniform float AmbientTintStrength;

in vec2 Frag_UV;
in vec4 Frag_Color;
out vec4 Out_Color;


void main()
{
	Out_Color = Frag_Color * texture(Texture, Frag_UV);

	if (Rim == 1) {
		float mixStrengthMin = 0.2;
		float mixStrength = mixStrengthMin;

		if (AmbientTint == 1) {
			mixStrength = AmbientTintStrength * 0.5;
			if (mixStrength < mixStrengthMin) {
				mixStrength = mixStrengthMin;
			}
		}

		vec2 uv = Frag_UV;
		uv = ((uv-0.5)*0.25) + 0.5;
		vec3 p = texture(Screen, uv).rgb;
		uv = ((uv-0.5)*0.25) + 0.5;
		p = mix(p, texture(Screen, uv).rgb, 0.5);
		uv = ((uv-0.5)*0.25) + 0.5;
		p = mix(p, texture(Screen, uv).rgb, 0.5);

		Out_Color.rgb = mix(Out_Color.rgb, Out_Color.rgb * p.rgb * Out_Color.a, mixStrength);

	} else if (AmbientTint == 1) {
		// everthing outside the rim is tinted blue if AmbientTint is true
		Out_Color.rgb = mix(Out_Color.rgb, Out_Color.rgb * vec3(0.0, 0.0, 1.0), AmbientTintStrength);
	}
	
	// add a small amount of perlin noise to the image
	if (Out_Color.a >= 1.0) {
		float perlin = fract(sin(dot(Frag_UV, vec2(12.9898, 78.233))*Time) * 43758.5453);
		perlin *= 0.03;
		Out_Color.rgb += vec3(perlin);
	}
}
