// majority of ideas taken from Mattias Gustavsson's crt-view. much of the
// implementation details are also from here.
//
//		https://github.com/mattiasgustavsson/crtview/
//
// other ideas taken from the crt-pi.glsl shader which is part of lib-retro:
//
//		https://github.com/libretro/glsl-shaders/blob/master/crt/shaders/crt-pi.glsl

uniform sampler2D Texture;
uniform sampler2D Frame;
in vec2 Frag_UV;
in vec4 Frag_Color;
out vec4 Out_Color;

uniform vec2 ScreenDim;
uniform int NumScanlines;
uniform int NumClocks;
uniform int Curve;
uniform int RoundedCorners;
uniform int Bevel;
uniform int Shine;
uniform int ShadowMask;
uniform int Scanlines;
uniform int Interference;
uniform int Noise;
uniform int Flicker;
uniform int Fringing;
uniform float CurveAmount;
uniform float RoundedCornersAmount;
uniform float BevelSize;
uniform float MaskIntensity;
uniform float MaskFine;
uniform float ScanlinesIntensity;
uniform float ScanlinesFine;
uniform float InterferenceLevel;
uniform float NoiseLevel;
uniform float FlickerLevel;
uniform float FringingAmount;
uniform float Time;

// rotation values are the values in hardware/television/specification/rotation.go
uniform int Rotation;

// Gold Noise taken from: https://www.shadertoy.com/view/ltB3zD
// Coprighted to dcerisano@standard3d.com not sure of the licence

// Gold Noise ©2015 dcerisano@standard3d.com
// - based on the Golden Ratio
// - uniform normalized distribution
// - fastest static noise generator function (also runs at low precision)
float PHI = 1.61803398874989484820459;  // Φ = Golden Ratio   
float gold_noise(in vec2 xy){
	return fract(tan(distance(xy*PHI, xy)*Time)*xy.x);
}

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
vec4 interferenceSmoothNoise1D(float x)
{
    float f0 = floor(x);
    float fr = fract(x);

    vec4 h0 = hash41( f0 );
    vec4 h1 = hash41( f0 + 1.0 );

    return h1 * fr + h0 * (1.0 - fr);
}


// From: ZX Spectrum SCREEN$ by Paul Malin
// https://www.shadertoy.com/view/ss3Xzj
vec4 interferenceNoise(vec2 uv)
{
    float scanLine = floor(uv.y * ScreenDim.y); 
    float scanPos = scanLine + uv.x;
	float timeSeed = fract( Time * 123.78 );
    
    return interferenceSmoothNoise1D( scanPos * 234.5 + timeSeed * 12345.6 );
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


void main() {
	// working on uv rather than Frag_UV for convenience and in case we need
	// Frag_UV unaltered later on for some reason
	vec2 uv = Frag_UV;

	// decrease size of image if bevel is active
	if (Bevel == 1) {
		uv = (uv - 0.5) * 1.02 + 0.5;
	}

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

	// capture the uv cordinates before reducing the size of the image. we use
	// this for the bevel effect which we don't want to be slightly larger than
	// the main image
	vec2 uv_bevel = uv;

	// reduce size of image (and the shine coordinates) shine if bevel is active
	if (Bevel == 1) {
		float margin = 1.0 + BevelSize;
		uv = (uv - 0.5) * margin + 0.5;
		shineUV = (shineUV - 0.5) * margin + 0.5;
	}

	// apply basic color
	vec4 Crt_Color = Frag_Color * texture(Texture, uv.st);

	// using y axis to determine scaling.
	float scaling = float(ScreenDim.y) / float(NumScanlines);

	// applying scanlines and/or shadowmask to the image causes it to dim
	const float brightnessCorrection = 0.7;

	// scanlines - only draw if scaling is large enough
	if (Scanlines == 1) {
		if (scaling > 1) {
			float scans = clamp(brightnessCorrection+ScanlinesIntensity*sin(uv.y*ScreenDim.y*ScanlinesFine), 0.0, 1.0);
			Crt_Color.rgb *= vec3(scans);
		} else {
			Crt_Color.rgb *= brightnessCorrection;
		}
	} else {
		Crt_Color.rgb *= brightnessCorrection;
	}

	// shadow mask - only draw if scaling is large enough
	if (ShadowMask == 1) {
		if (scaling > 1) {
			float mask = clamp(brightnessCorrection+MaskIntensity*sin(uv.x*ScreenDim.y*MaskFine), 0.0, 1.0);
			Crt_Color.rgb *= vec3(mask);
		} else {
			Crt_Color.rgb *= brightnessCorrection;
		}
	} else {
		Crt_Color.rgb *= brightnessCorrection;
	}

	// RF Interference
	if (Interference == 1) {
		// interferencw
		vec4 noise = interferenceNoise(uv);
		uv.x += noise.w * InterferenceLevel / 150;
	}

	// noise 
	if (Noise == 1) {
		float n;
		n = gold_noise(gl_FragCoord.xy);
		if (n < 0.33) {
			Crt_Color.r *= max(1.0-NoiseLevel, n);
		} else if (n < 0.66) {
			Crt_Color.g *= max(1.0-NoiseLevel, n);
		} else {
			Crt_Color.b *= max(1.0-NoiseLevel, n);
		}

	}

	// flicker
	if (Flicker == 1) {
		Crt_Color *= (1.0-FlickerLevel*(sin(50.0*Time+uv.y*2.0)*0.5+0.5));
	}

	// fringing (chromatic aberration)
	if (Fringing == 1) {
		vec2 ab = vec2(0.0);

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
		Crt_Color.rgb *= 0.50;
	} else {
		// adjust brightness if fringing hasn't been applied
		Crt_Color.rgb *= 1.50;
	}

	// shine affect
	if (Shine == 1) {
		vec2 shineUV = (shineUV - 0.5) * 1.2 + 0.5;
		float shine = (1.0-shineUV.s)*(1.0-shineUV.t);
		Crt_Color = mix(Crt_Color, vec4(1.0), shine*0.05);
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

	// bevel
	if (Bevel == 1) {
		vec2 bl = smoothstep(vec2(-BevelSize), vec2(BevelSize), uv_bevel.st);
		vec2 tr = smoothstep(vec2(-BevelSize), vec2(BevelSize), 1.0-uv_bevel.st);
		float pct = bl.x * bl.y * tr.x * tr.y;
		if (pct < 0.75) {
			Crt_Color = vec4(0.1, 0.1, 0.11, 1.0) * (1.0-pct);
		}
	}

	// finalise color
	Out_Color = Crt_Color;
}
