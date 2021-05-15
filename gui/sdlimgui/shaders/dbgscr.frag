#version 150

uniform int ShowCursor;  
uniform int IsCropped; 
uniform vec2 ScreenDim;
uniform float ScalingX;
uniform float ScalingY;
uniform float LastX; 
uniform float LastY;
uniform float Hblank;
uniform float TopScanline;
uniform float BotScanline;
uniform float OverlayAlpha;

uniform sampler2D Texture;

in vec2 Frag_UV;
in vec4 Frag_Color;
out vec4 Out_Color;

bool isNearEqual(float x, float y, float epsilon)
{
	return abs(x - y) <= epsilon;
}

const float cursorSize = 1.0;

vec2 coords = Frag_UV.xy;

void main()
{
	Out_Color = Frag_Color * texture(Texture, Frag_UV.st);

	// value of one pixel
	float pixelX = 1.0 / ScreenDim.x;
	float pixelY = 1.0 / ScreenDim.y;

	// the size of one texel (used for painting and cursor positioning)
	float texelX = pixelX * ScalingX;
	float texelY = pixelY * ScalingY;

	// adjusted last x/y coordinates. lastY depends on IsCropped
	float lastX = pixelX * LastX;
	float lastY = pixelY * LastY;

	// screen boundaries. depends on IsCropped
	float topBoundary;
	float botBoundary;

	if (IsCropped == 1) {
		topBoundary = 0;
		botBoundary = (BotScanline - TopScanline) / ScreenDim.y;
		lastY -=  pixelY * TopScanline;
	} else {
		float hblank = pixelX * Hblank;
		topBoundary = pixelY * TopScanline;
		botBoundary = pixelY * BotScanline;

		// screen guides
		if (isNearEqual(coords.x, hblank-pixelX, pixelX) ||
		   isNearEqual(coords.y, topBoundary-pixelY, pixelY) ||
		   isNearEqual(coords.y, botBoundary+pixelY, pixelY)) {
			Out_Color.r = 0.5;
			Out_Color.g = 0.5;
			Out_Color.b = 1.0;
			Out_Color.a = 0.1;
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
			Out_Color.g = 0.0;
			Out_Color.b = 0.0;
			Out_Color.a = 1.0;
			return;
		}

		// for cropped screens there are a few more conditions that we need to
		// consider for drawing an off-screen cursor
		if (IsCropped == 1) {
			// when VBLANK is active but HBLANK is off
			if (isNearEqual(coords.x, lastX, cursorSize * texelX/2)) {
				// top of screen
				if (lastY < 0 && isNearEqual(coords.y, 0, cursorSize*texelY)) {
					Out_Color.r = 1.0;
					Out_Color.g = 0.0;
					Out_Color.b = 0.0;
					Out_Color.a = 1.0;
					return;
				}
			
				// bottom of screen (knocking a pixel off the scanline
				// boundary check to make sure the cursor is visible)
				if (lastY > botBoundary-pixelY && isNearEqual(coords.y, botBoundary, cursorSize*texelY)) {
					Out_Color.r = 1.0;
					Out_Color.g = 0.0;
					Out_Color.b = 0.0;
					Out_Color.a = 1.0;
					return;
				}
			}

			// when HBLANK and VBLANK are both active
			if (lastX < 0 && isNearEqual(coords.x, 0, cursorSize*texelX/2)) {
				// top/left corner of screen
				if (lastY < 0 && isNearEqual(coords.y, 0, cursorSize*texelY)) {
					Out_Color.r = 1.0;
					Out_Color.g = 0.0;
					Out_Color.b = 0.0;
					Out_Color.a = 1.0;
					return;
				}

				// bottom/left corner of screen (knocking a pixel off the
				// scanline boundary check to make sure the cursor is
				// visible)
				if (lastY > botBoundary-pixelY && isNearEqual(coords.y, botBoundary, cursorSize*texelY)) {
					Out_Color.r = 1.0;
					Out_Color.g = 0.0;
					Out_Color.b = 0.0;
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
				// only affect pixels with an active alpha channel
				if (Out_Color.a != 0.0) {
					// wash out color and mix with original pixel. this will
					// preseve the anti-aliased curved CRT effect if it's
					// present. the more naive "Out_Color.a = 0.5;" will cause
					// and ugly edge to the screen.
					vec4 c = Out_Color;
					c.a = 0.0;
					Out_Color = mix(Out_Color, c, 0.5);
				}
				return;
			}
		}
	}
}

