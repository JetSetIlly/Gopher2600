#version 150

const int GUI = 0;
const int DebugScr = 1;
const int Overlay = 2;
const int PlayScr = 3;
const int PrefsCRT = 4;

const int True = 1;
const int False = 0;

precision mediump float;

// the type of rendering to be performad
uniform int ImageType;

uniform int ShowCursor;  
uniform int IsCropped; 
uniform vec2 ScreenDim;
uniform vec2 UncroppedScreenDim;
uniform float ScalingX;
uniform float ScalingY;
uniform float LastX; 
uniform float LastY;
uniform float Hblank;
uniform float TopScanline;
uniform float BotScanline;

uniform int EnableCRT;
uniform int EnablePhosphor;
uniform int EnableShadowMask;
uniform int EnableScanlines;
uniform int EnableNoise;
uniform int EnableBlur;
uniform int EnableVignette;
uniform float PhosphorSpeed;
uniform float MaskBrightness;
uniform float ScanlinesBrightness;
uniform float NoiseLevel;
uniform float BlurLevel;
uniform float RandSeed;

uniform sampler2D Texture;
uniform sampler2D PhosphorTexture;

in vec2 Frag_UV;
in vec4 Frag_Color;
out vec4 Out_Color;


bool isNearEqual(float x, float y, float epsilon)
{
	return abs(x - y) <= epsilon;
}

const float cursorSize = 1.0;

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

float relativeLuminance(vec3 rgb) {
	return (float(rgb.r) / 255 * 0.2126) + (float(rgb.g) / 255 * 0.7152) + (float(rgb.b) / 255 * 0.0722);
}

vec2 coords = Frag_UV.xy;

float hblank;
float topScanline;
float botScanline;

// the size of one texel (used for painting and cursor positioning)
float texelX;
float texelY;

// basic CRT effects
// -----------------
// some ideas taken from the crt-pi.glsl shader which is part of lib-retro
//
// https://github.com/libretro/glsl-shaders/blob/master/crt/shaders/crt-pi.glsl
void crt() {
	vec4 Crt_Color = Frag_Color * texture(Texture, Frag_UV.st);

	// phosphor
	if (EnablePhosphor == True) {
		if (Crt_Color.r == 0 && Crt_Color.g == 0 && Crt_Color.b == 0) {
			vec4 ph = texture(PhosphorTexture, vec2(coords.x, coords.y)).rgba;
			Crt_Color.rgb = ph.rgb;
			Crt_Color.r *= pow(ph.a, 145*PhosphorSpeed);
			Crt_Color.g *= pow(ph.a, 160*PhosphorSpeed);
			Crt_Color.b *= pow(ph.a, 130*PhosphorSpeed);
		}
	}

	// noise
	if (EnableNoise == True) {
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

	vec2 grid = vec2(floor(gl_FragCoord.x), floor(gl_FragCoord.y));

	// shadow masking
	if (EnableShadowMask == True) {
		if (mod(grid.y, 2) == 0.0) {
			Crt_Color.a *= MaskBrightness;
		}
	}

	// scanline effect
	if (EnableScanlines == True) {
		if (mod(grid.x, 2) == 0.0) {
			Crt_Color.a *= ScanlinesBrightness;
		}
	}

	// blur
	if (EnableBlur == True) {
		float bx = texelX*BlurLevel;
		float by = texelY*BlurLevel;
		if (coords.x-bx > 0.0 && coords.x+bx < 1.0 && coords.y-by > 0.0 && coords.y+by < 1.0) {
			Crt_Color.r += texture(Texture, vec2(coords.x-bx, coords.y+by)).r;
			Crt_Color.g += texture(Texture, vec2(coords.x+bx, coords.y-by)).g;
			Crt_Color.b += texture(Texture, vec2(coords.x+bx, coords.y+by)).b;
			Crt_Color.rgb *= 0.50;
		}
	}

	// vignette effect
	//
	// in the case of the CRT Prefs preview screen the vignette is applied to
	// the visible area. it is not applied to the entirity of the screen as you
	// might expect. this is a happy accident. I think it's nice to see a
	// representation of the effect in it's entirety.
	if (EnableVignette == True) {
		float vignette;

		if (IsCropped == True) {
			vignette = (10.0*coords.x*coords.y*(1.0-coords.x)*(1.0-coords.y));

		} else {
			// f is used to factor the vignette value. In the "cropped" branch we
			// use a factor value of 10. to visually mimic the vignette effect a
			// value of about 25 is required (using Pitfall as a template). I don't
			// understand this well enough to say for sure what the relationship
			// between 25 and 10 is, but the following ratio between
			// cropped/uncropped widths gives us a value of 23.5
			float f =UncroppedScreenDim.x/(UncroppedScreenDim.x-ScreenDim.x);
			vignette = (f*(coords.x-hblank)*(coords.y-topScanline)*(1.0-coords.x)*(1.0-coords.y));
		}

		Crt_Color.rgb *= pow(vignette, 0.10) * 1.2;
	}

	Out_Color = Crt_Color;
}

void debugscr() {
	float lastX;
	float lastY;

	// pixels are texels without the scaling applied
	float pixelX;
	float pixelY;

	if (IsCropped == True) {
		texelX = ScalingX / ScreenDim.x;
		texelY = ScalingY / ScreenDim.y;
		hblank = Hblank / ScreenDim.x;
		lastX = LastX / ScreenDim.x;
		topScanline = 0;
		botScanline = (BotScanline - TopScanline) / ScreenDim.y;

		// the LastY coordinate refers to the full-frame scanline. the cropped
		// texture however counts from zero at the visible edge so we need to
		// adjust the lastY value by the TopScanline value.
		//
		// note that there's no need to do this for LastX because the
		// horizontal position is counted from -68 in all instances.
		lastY = (LastY - TopScanline) / ScreenDim.y;
	} else {
		texelX = ScalingX / UncroppedScreenDim.x;
		texelY = ScalingY / UncroppedScreenDim.y;
		hblank = Hblank / UncroppedScreenDim.x;
		topScanline = TopScanline / UncroppedScreenDim.y;
		botScanline = BotScanline / UncroppedScreenDim.y;
		lastX = LastX / UncroppedScreenDim.x;
		lastY = LastY / UncroppedScreenDim.y;
	}

	pixelX = texelX / ScalingX;
	pixelY = texelY / ScalingY;

	// if the entire frame is being shown then plot the screen guides
	if (IsCropped == False) {
		if (isNearEqual(coords.x, hblank, pixelX) ||
		   isNearEqual(coords.y, topScanline, pixelY) ||
		   isNearEqual(coords.y, botScanline, pixelY)) {
			Out_Color.r = 1.0;
			Out_Color.g = 1.0;
			Out_Color.b = 1.0;
			Out_Color.a = 0.2;
			return;
		}
	}

	// when ShowCursor is true then there is some additional processing we need to perform
	if (ShowCursor == 1) {
		// draw cursor if pixel is at the last x/y position
		if (lastY >= 0 && lastX >= 0) {
			if (isNearEqual(coords.y, lastY+texelY, cursorSize*texelY) && isNearEqual(coords.x, lastX+texelX, cursorSize*texelX/2)) {
				Out_Color.r = 1.0;
				Out_Color.g = 1.0;
				Out_Color.b = 1.0;
				Out_Color.a = 1.0;
				return;
			}
		}

		// draw off-screen cursor for HBLANK
		if (lastX < 0 && isNearEqual(coords.y, lastY+texelY, cursorSize*texelY) && isNearEqual(coords.x, 0, cursorSize*texelX/2)) {
			Out_Color.r = 1.0;
			Out_Color.a = 1.0;
			return;
		}

		// for cropped screens there are a few more conditions that we need to
		// consider for drawing an off-screen cursor
		if (IsCropped == True) {
			// when VBLANK is active but HBLANK is off
			if (isNearEqual(coords.x, lastX, cursorSize * texelX/2)) {
				// top of screen
				if (lastY < 0 && isNearEqual(coords.y, 0, cursorSize*texelY)) {
					Out_Color.r = 1.0;
					Out_Color.a = 1.0;
					return;
				}
			
				// bottom of screen (knocking a pixel off the scanline
				// boundary check to make sure the cursor is visible)
				if (lastY > botScanline-pixelY && isNearEqual(coords.y, botScanline, cursorSize*texelY)) {
					Out_Color.r = 1.0;
					Out_Color.a = 1.0;
					return;
				}
			}

			// when HBLANK and VBLANK are both active
			if (lastX < 0 && isNearEqual(coords.x, 0, cursorSize*texelX/2)) {
				// top/left corner of screen
				if (lastY < 0 && isNearEqual(coords.y, 0, cursorSize*texelY)) {
					Out_Color.r = 1.0;
					Out_Color.a = 1.0;
					return;
				}

				// bottom/left corner of screen (knocking a pixel off the
				// scanline boundary check to make sure the cursor is
				// visible)
				if (lastY > botScanline-pixelY && isNearEqual(coords.y, botScanline, cursorSize*texelY)) {
					Out_Color.r = 1.0;
					Out_Color.a = 1.0;
					return;
				}
			}
		}

		// painting effect draws pixels with faded alpha if lastX and lastY
		// are less than rendering coords.
		//
		// as a special case, we ignore the first scanline and do not fade the
		// previous image on a brand new frame. note that we're using the
		// unadjusted LastY value for this
		if (LastY > 0) {
			if (coords.y > lastY+texelY || (isNearEqual(coords.y, lastY+texelY, texelY) && coords.x > lastX+texelX)) {
				Out_Color = Frag_Color * texture(Texture, Frag_UV.st);
				Out_Color.a = 0.5;
				return;
			}
		}
	}

	// only apply CRT effects on the "cropped" area of the screen. we can think
	// of the cropped area as the "play" area
	if (EnableCRT == True && !(IsCropped == False && (coords.x < hblank || coords.y < topScanline || coords.y > botScanline))) {
		crt();
		return;
	}

	Out_Color = Frag_Color * texture(Texture, Frag_UV.st);
}

void playscr() {
	texelX = ScalingX / ScreenDim.x;
	texelY = ScalingY / ScreenDim.y;
	if (EnableCRT == True) {
		crt();
		return;
	}
	Out_Color = Frag_Color * texture(Texture, Frag_UV.st);
}

void prefscrt() {
	texelX = ScalingX / ScreenDim.x;
	texelY = ScalingY / ScreenDim.y;
	crt();
}

void overlay() {
	Out_Color = Frag_Color * texture(Texture, Frag_UV.st);
}

void imgui() {
	Out_Color = vec4(Frag_Color.rgb, Frag_Color.a * texture(Texture, Frag_UV.st).r);
}

void main()
{
	switch (ImageType) {
	case GUI:
		imgui();
		break;
	case Overlay:
		overlay();
		break;
	case DebugScr:
		debugscr();
		break;
	case PlayScr:
		playscr();
		break;
	case PrefsCRT:
		prefscrt();
		break;
	}
} 

