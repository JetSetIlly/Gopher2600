#version 150

// majority of ideas taken from Mattias Gustavsson's crt-view. much of the
// implementation details are also from here:
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
uniform int Curve;
uniform int ShadowMask;
uniform int Scanlines;
uniform int Noise;
uniform int Fringing;
uniform int Flicker;
uniform float CurveAmount;
uniform float MaskBright;
uniform float ScanlinesBright;
uniform float NoiseLevel;
uniform float FringingAmount;
uniform float FlickerLevel;
uniform float Time;


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

// taken directly from https://github.com/mattiasgustavsson/crtview/
vec2 curve(in vec2 uv)
{
	uv = (uv - 0.5) * 2.1;
	uv *= 1.1;	
	uv.x *= 1.0 + pow((abs(uv.y) / 5.0), 2.0);
	uv.y *= 1.0 + pow((abs(uv.x) / 4.0), 2.0);
	uv  = (uv / 2.0) + 0.5;
	uv =  uv * 0.92 + 0.04;
	return uv;
}

void main() {
	vec4 Crt_Color;
	vec2 uv = Frag_UV;

	if (Curve == 1) {
		// curve UV coordinates. 
		float m = (CurveAmount * 0.4) + 0.6; // bring into sensible range
		uv = mix(curve(uv), uv, m);
	}

	// after this point every UV reference is to the curved UV

	// basic color
	Crt_Color = Frag_Color * texture(Texture, uv.st);

	// video-black correction
	if (Curve == 1) {
		float vb = 0.16;
		Crt_Color.rgb = clamp(Crt_Color.rgb, vec3(vb), vec3(1.0));
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

	// shadow mask
	if (ShadowMask == 1) {
		if (mod(floor(gl_FragCoord.x), 2) == 0.0) {
			Crt_Color.rgb *= MaskBright;
		}
	}

	// scanlines
	if (Scanlines == 1) { 
		float scans = clamp(0.35+0.18*sin(uv.y*ScreenDim.y*2.0), 0.0, 1.0);
		float s = pow(scans,1.0-ScanlinesBright);
		Crt_Color.rgb *= vec3(s);
	}

	// fringing (chromatic aberration)
	vec2 ab = vec2(0.0);
	if (Fringing == 1) {
		if (Curve == 1) {
			ab.x = abs(uv.x-0.5);
			ab.y = abs(uv.y-0.5);

			// modulate fringing amount by curvature
			float m = 0.020 - (0.010 * CurveAmount);
			float l = FringingAmount * m;

			// aberration amount limited to reasonabl values
			ab = clamp(ab*l, 0.0009, 0.004);
		} else {
			float f = FringingAmount * 0.005;
			ab = vec2(f);
		}
	}

	// always perform the aberration if the ab amount is 0.0. without this and
	// if Fringing is off, the screen is too harsh.
	Crt_Color.r += texture(Texture, vec2(uv.x-ab.x, uv.y+ab.y)).r;
	Crt_Color.g += texture(Texture, vec2(uv.x+ab.x, uv.y-ab.y)).g;
	Crt_Color.b += texture(Texture, vec2(uv.x+ab.x, uv.y+ab.y)).b;
	Crt_Color.rgb *= 0.50;

	// vignette effect
	if (Curve == 1) {
		float vignette = 10.0*uv.x*uv.y*(1.0-uv.x)*(1.0-uv.y);
		Crt_Color.rgb *= pow(vignette, 0.10) * 1.3;
	}

	// finalise color
	Out_Color = Crt_Color;
}
