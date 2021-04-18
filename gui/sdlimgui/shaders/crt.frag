#version 150

uniform sampler2D Texture;
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
uniform float RandSeed;

// Gold Noise taken from: https://www.shadertoy.com/view/ltB3zD
// Coprighted to dcerisano@standard3d.com not sure of the licence

// Gold Noise ©2015 dcerisano@standard3d.com
// - based on the Golden Ratio
// - uniform normalized distribution
// - fastest static noise generator function (also runs at low precision)
float PHI = 1.61803398874989484820459;  // Φ = Golden Ratio   
float gold_noise(in vec2 xy){
	return fract(tan(distance(xy*PHI, xy)*RandSeed)*xy.x);
}

vec2 coords = Frag_UV.xy;

// basic CRT effects
// -----------------
// some ideas taken from the crt-pi.glsl shader which is part of lib-retro:
//
// https://github.com/libretro/glsl-shaders/blob/master/crt/shaders/crt-pi.glsl
//
// also from Mattias Gustavsson's crt-view:
//
// https://github.com/mattiasgustavsson/crtview/
void main() {
	vec4 Crt_Color = Frag_Color * texture(Texture, Frag_UV.st);

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

	// shadow masking and scanlines
	float shadowWeight = 1.0;
	vec2 grid = vec2(floor(gl_FragCoord.x), floor(gl_FragCoord.y));
	if (ShadowMask == 1) {
		if (mod(grid.x, 2) == 0.0) {
			Crt_Color.rgb *= MaskBrightness*shadowWeight;
		}
	}
	if (Scanlines == 1) {
		if (mod(grid.y, 2) == 0.0) {
			Crt_Color.rgb *= ScanlinesBrightness*shadowWeight;
		}
	}

	// blur
	if (Blur == 1) {
		float texelX = ScalingX / ScreenDim.x;
		float texelY = ScalingY / ScreenDim.y;
		float bx = texelX*BlurLevel;
		float by = texelY*BlurLevel;
		if (coords.x-bx > 0.0 && coords.x+bx < 1.0 && coords.y-by > 0.0 && coords.y+by < 1.0) {
			Crt_Color.r += texture(Texture, vec2(coords.x-bx, coords.y+by)).r;
			Crt_Color.g += texture(Texture, vec2(coords.x+bx, coords.y-by)).g;
			Crt_Color.b += texture(Texture, vec2(coords.x+bx, coords.y+by)).b;
			Crt_Color.rgb *= 0.50;
		}
	}

	// flicker
	if (Flicker == 1) {
		Crt_Color *= (1.0-FlickerLevel*(sin(50.0*RandSeed+Frag_UV.y*2.0)*0.5+0.5));
	}

	// vignette effect
	//
	// in the case of the CRT Prefs preview screen the vignette is applied to
	// the visible area. it is not applied to the entirity of the screen as you
	// might expect. this is a happy accident. I think it's nice to see a
	// representation of the effect in it's entirety.
	if (Vignette == 1) {
		float vignette;
		vignette = (10.0*coords.x*coords.y*(1.0-coords.x)*(1.0-coords.y));
		Crt_Color.rgb *= pow(vignette, 0.10) * 1.2;
	}

	Out_Color = Crt_Color;
}
