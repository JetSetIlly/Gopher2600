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
uniform float ScalingX;
uniform float ScalingY;

uniform int ShadowMask;
uniform int Scanlines;
uniform int Noise;
uniform int Blur;
uniform int Vignette;
uniform int Flicker;
uniform float MaskBrightness;
uniform float ScanlinesBrightness;
uniform float NoiseLevel;
uniform float BlurLevel;
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
vec2 curve(vec2 uv)
{
	uv = (uv - 0.5) * 2.0;
	uv *= 1.1;	
	uv.x *= 1.0 + pow((abs(uv.y) / 5.0), 2.0);
	uv.y *= 1.0 + pow((abs(uv.x) / 4.0), 2.0);
	uv  = (uv / 2.0) + 0.5;
	uv =  uv *0.92 + 0.04;
	return uv;
}

void main() {
	vec4 Crt_Color;
	vec2 uv = Frag_UV;

	// curve. taken from https://github.com/mattiasgustavsson/crtview/
	uv = mix(curve(uv), uv, 0.8);

	// after this point every UV reference is to the curved UV

	// basic color
	Crt_Color = Frag_Color * texture(Texture, uv.st);

	// correct video-black
	Crt_Color.rgb = clamp(Crt_Color.rgb, vec3(0.115,0.115,0.115), vec3(1.0,1.0,1.0));

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

	// shadow masking and scanlines
	vec2 grid = vec2(floor(gl_FragCoord.x), floor(gl_FragCoord.y));
	if (ShadowMask == 1) {
		if (mod(grid.x, 2) == 0.0) {
			Crt_Color.rgb *= MaskBrightness;
		}
	}
	if (Scanlines == 1) {
		if (mod(grid.y, 4) == 0.0) {
			Crt_Color.rgb *= ScanlinesBrightness;
		}
	}

	// chromatic aberation
	float blurLevel = BlurLevel;
	if (Blur == 0) {
		blurLevel = 0.00;
	}
	float texelX = ScalingX / ScreenDim.x;
	float texelY = ScalingY / ScreenDim.y;
	float bx = texelX*blurLevel;
	float by = texelY*blurLevel;
	if (uv.x-bx > 0.0 && uv.x+bx < 1.0 && uv.y-by > 0.0 && uv.y+by < 1.0) {
		Crt_Color.r += texture(Texture, vec2(uv.x-bx, uv.y+by)).r;
		Crt_Color.g += texture(Texture, vec2(uv.x+bx, uv.y-by)).g;
		Crt_Color.b += texture(Texture, vec2(uv.x+bx, uv.y+by)).b;
		Crt_Color.rgb *= 0.50;
	}

	// vignette effect
	if (Vignette == 1) {
		// scaling and adjusting the uv use to apply the vinette to make sure
		// we cover up the extreme edges of the curvature.
		vec2 scuv = uv;
		scuv.y *= 1.001;
		scuv.x *= 1.005;
		scuv.x -= 0.004;

		float vignette = 10.0*scuv.x*scuv.y*(1.0-scuv.x)*(1.0-scuv.y);
		Crt_Color.rgb *= pow(vignette, 0.10) * 1.3;
	}

	// clamp
	if (uv.x < 0.00 || uv.x > 1.0) {
		Crt_Color *= 0.0;
	}
	if (uv.y < 0.0 || uv.y > 1.0) {
		Crt_Color *= 0.0;
	}

	Out_Color = Crt_Color;
}
