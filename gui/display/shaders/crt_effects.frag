// majority of ideas taken from Mattias Gustavsson's crt-view. much of the
// implementation details are also from here.
//
//		https://github.com/mattiasgustavsson/crtview/
//
// other ideas taken from the crt-pi.glsl shader which is part of lib-retro:
//
//		https://github.com/libretro/glsl-shaders/blob/master/crt/shaders/crt-pi.glsl

uniform sampler2D Texture;
in vec2 Frag_UV;
in vec4 Frag_Color;
out vec4 Out_Color;

uniform vec2 ScreenDim;
uniform int NumScanlines;
uniform int NumClocks;
uniform int Curve;
uniform int RoundedCorners;
uniform int Shine;
uniform int ShadowMask;
uniform int Scanlines;
uniform int Interference;
uniform int Fringing;
uniform float BlackLevel;
uniform float CurveAmount;
uniform float RoundedCornersAmount;
uniform float BevelSize;
uniform float MaskIntensity;
uniform float ScanlinesIntensity;
uniform float InterferenceLevel;
uniform float FringingAmount;
uniform float Time;

// rotation values are the values in hardware/television/specification/rotation.go
uniform int Rotation;

// the screenshot boolean indicates that the shader is working to create a still
// image. this affects the intensity of some effects
uniform int Screenshot;


// From: Hash without Sine by Dave Hoskins
// https://www.shadertoy.com/view/4djSRW
vec4 hash41(float p)
{
	vec4 p4 = fract(vec4(p) * vec4(.1031, .1030, .0973, .1099));
    p4 += dot(p4, p4.wzxy+33.33);
    return fract((p4.xxyz+p4.yzzw)*p4.zywx);    
}

// From: ZX Spectrum SCREEN$ by Paul Malin
// https://www.shadertoy.com/view/ss3Xzj
vec4 interferenceSmoothAmount(float x)
{
    float f0 = floor(x);
    float fr = fract(x);

    vec4 h0 = hash41( f0 );
    vec4 h1 = hash41( f0 + 1.0 );

    return h1 * fr + h0 * (1.0 - fr);
}


// From: ZX Spectrum SCREEN$ by Paul Malin
// https://www.shadertoy.com/view/ss3Xzj
vec4 interferenceAmount(vec2 uv)
{
    float scanLine = floor(uv.y * ScreenDim.y); 
    float scanPos = scanLine + uv.x;
	float timeSeed = fract( Time * 123.78 );
    
    return interferenceSmoothAmount( scanPos * 234.5 + timeSeed * 12345.6 );
}


// taken directly from https://github.com/mattiasgustavsson/crtview/
vec2 curve(in vec2 uv)
{
	uv = (uv - 0.5) * 2.31;
	uv.x *= 1.0 + pow((abs(uv.y) / 5.0), 2.0);
	uv.y *= 1.0 + pow((abs(uv.x) / 4.0), 2.0);
	uv  = (uv / 2.0) + 0.5;
	uv =  uv * 0.92 + 0.04;
	return uv;
}

// convert RGB to YIQ. note that the Y component of the YIQ vector is
// actually the x field of the vectore and not y!
vec3 RGBtoYIQ(in vec3 rgb)
{
	float gamma = 1.0/2.2;

	mat3 adjust = mat3(
		vec3(gamma, 0.0, 0.0),
		vec3(0.0, gamma, 0.0),
		vec3(0.0, 0.0, gamma)
	);

	adjust *= mat3(
		vec3(0.299, 0.587, 0.114),
		vec3(0.5959, -0.2746, -0.3213),
		vec3(0.2115, -0.5227, 0.3112)
	);

	return rgb * adjust;
}

vec3 YIQtoRGB(in vec3 yiq)
{
	mat3 adjust = mat3(
		vec3(1, 0.956, 0.619),
		vec3(1, -0.272, -0.647),
		vec3(1, -1.106, 1.703)
	);

	float gamma = 2.2;

	adjust *= mat3(
		vec3(gamma, 0.0, 0.0),
		vec3(0.0, gamma, 0.0),
		vec3(0.0, 0.0, gamma)
	);

	return yiq * adjust;
}

void main() {
	// working on uv rather than Frag_UV for convenience and in case we need
	// Frag_UV unaltered later on for some reason
	vec2 uv = Frag_UV;

	// apply curve. uv coordinates will be curved from here on out
	if (Curve == 1) {
		float m = (CurveAmount * 0.4) + 0.6; // bring into sensible range
		uv = mix(curve(uv), uv, m);
	}

	// capture the uv coordinates before rotation. we use this for the shine
	// effect, which we don'e want rotated
	vec2 shineUV = uv;

	// apply rotation
	float textureRatio = ScreenDim.x / ScreenDim.y;
	float rads = 1.5708 * Rotation;
	uv -= 0.5;
    uv = vec2(
        cos(rads) * uv.x + sin(rads) * uv.y,
        cos(rads) * uv.y - sin(rads) * uv.x
    );
	uv += 0.5;

	// apply basic color
	vec4 Crt_Color = Frag_Color * texture(Texture, uv.st);

	// black correction 
	Crt_Color.rgb = clamp(Crt_Color.rgb, vec3(BlackLevel*3.00), Crt_Color.rgb);

	// the following effects are applied to the YIQ signal
	vec3 yiq = RGBtoYIQ(Crt_Color.rgb);

	// Interference. This effect is split into two halves. The second half happens later
	if (Interference == 1) {
		vec4 noise = interferenceAmount(uv);

		// a little but of horizontal movement works well
		uv.x += noise.w * InterferenceLevel / 150;

		// YIQ interference
		yiq.x *= 0.85;
		yiq.x *= 0.5 + noise.w * InterferenceLevel * 2.0;
	} else {
		// no interference but we want to reduce the Y channel so that the
		// apparent brightness of the image is similar to when interference is
		// enabled
		yiq.x *= 0.60;
	}

	// scanline/mask effect
	{
		// using y axis to determine scaling.
		float scaling = float(ScreenDim.y) / float(NumScanlines);


		// scanlines - only draw if scaling is large enough
		if (Scanlines == 1 && scaling > 2) {
			float scans = clamp(1.0+ScanlinesIntensity*sin(uv.y*NumScanlines*5), 0.0, 1.0);
			yiq.x *= scans;
		}

		// shadow mask - only draw if scaling is large enough
		if (ShadowMask == 1 && scaling > 2) {
			float mask = clamp(1.0+MaskIntensity*sin(uv.x*NumClocks*8), 0.0, 1.0);
			yiq.x *= mask;
		}

	}

	// the end of the effects that work on the YIQ signal
	Crt_Color.rgb = YIQtoRGB(yiq);

	// colour fringing (chromatic aberration). we always do this even if
	// fringing is disabled, we just use an aberation value of zero. performing
	// if seems to soften the scanlines / shadow mask, which is desirable
	{
		vec2 ab = vec2(0.0);
		if (Fringing == 1) {
			if (Curve == 1) {
				ab.x = abs(uv.x-0.5);
				ab.y = abs(uv.y-0.5);

				// modulate fringing amount by curvature
				float m = 0.020 - (0.010 * CurveAmount);
				float l = FringingAmount * m;

				// aberration amount limited to reasonable values
				ab *= l;

				// minimum amount of aberration
				ab = clamp(vec2(0.04), ab, ab);
			} else {
				ab.x = abs(uv.x-0.5);
				ab.y = abs(uv.y-0.5);
				ab *= FringingAmount * 0.015;
			}
		}

		// adjust sign depending on which quadrant the pixel is in
		if (uv.x <= 0.5) {
			ab.x *= -1;
		}
		if (uv.y <= 0.5) {
			ab.y *= -1;
		}

		// perform the aberration
		Crt_Color.r += texture(Texture, vec2(uv.x+(1.0*ab.x), uv.y+(1.0*ab.y))).r;
		Crt_Color.g += texture(Texture, vec2(uv.x+(1.4*ab.x), uv.y+(1.4*ab.y))).g;
		Crt_Color.b += texture(Texture, vec2(uv.x+(1.8*ab.x), uv.y+(1.8*ab.y))).b;

		// decrease gain slightly caused by aberration 
		Crt_Color.rgb *= 0.70;
	}

	// we always add perlin noise because we always want to elimiate banding. we
	// used to use perlin for interference noise but we've found it better to
	// give a small random deviation to the Y channel
	float perlin = fract(sin(dot(uv, vec2(12.9898, 78.233))*Time) * 43758.5453);
	perlin *= 0.005;
	Crt_Color.rgb += vec3(perlin);

	// shine effect
	if (Shine == 1) {
		vec2 shineUV = 1.0 - shineUV;
		float shine = shineUV.s * shineUV.t;
		Crt_Color.rgb = mix(Crt_Color.rgb, vec3(1.0), shine*0.05);
	}

	// vignette effect
	if (Curve == 1) {
		float vignette = 10*uv.x*uv.y*(1.0-uv.x)*(1.0-uv.y);
		Crt_Color.rgb *= pow(vignette, 0.12) * 1.1;
	}

	// rounded corners
	if (RoundedCorners == 1) {
		float margin = RoundedCornersAmount / 4.0;
		vec2 bl = smoothstep(vec2(-margin), vec2(RoundedCornersAmount)*1.1, uv.st);
		vec2 tr = smoothstep(vec2(-margin), vec2(RoundedCornersAmount)*1.1, 1.0-uv.st);
		float pct = bl.x * bl.y * tr.x * tr.y;
		if (pct < 0.1) {
			Crt_Color = vec4(pct);
		}
	}

	// finalise color
	Out_Color = Crt_Color;
}
