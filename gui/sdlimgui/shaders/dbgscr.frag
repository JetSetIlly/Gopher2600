#version 150

uniform int ShowCursor;  
uniform vec2 ScreenDim;
uniform float ScalingX;
uniform float ScalingY;
uniform float LastX; 
uniform float LastY;
uniform float Hblank;
uniform float OverlayAlpha;
uniform float LastNewFrameAtScanline;

uniform int IsCropped; 

// the top abd bottom scanlines to show. in the case of IsCropped then these
// values will be used to draw the screen guides
uniform float VisibleTop;
uniform float VisibleBottom;

// the number of scanlines in the uncropped display
uniform float ScanlinesTotal;

uniform sampler2D Texture;

in vec2 Frag_UV;
in vec4 Frag_Color;
out vec4 Out_Color;

bool isNearEqual(float x, float y, float epsilon)
{
	return abs(x - y) <= epsilon;
}

const float cursorSize = 1.0;

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

	// bottom screen boundary. depends on IsCropped
	float visibleBottom;

	if (IsCropped == 1) {
		visibleBottom = (VisibleBottom - VisibleTop) / ScreenDim.y;
		lastY -=  pixelY * VisibleTop;
	} else {
		// top/bottom guides
		float visibleTop = pixelY * VisibleTop;
		visibleBottom = pixelY * VisibleBottom;
		if (isNearEqual(Frag_UV.y, visibleTop-pixelY, pixelY) || isNearEqual(Frag_UV.y, visibleBottom+pixelY, pixelY)) {
			if (mod(floor(gl_FragCoord.x), 4) < 2.0) {
				Out_Color.r = 1.0;
				Out_Color.g = 1.0;
				Out_Color.b = 1.0;
				Out_Color.a = 0.1;
				return;
			}
		}

		// hblank guide
		float hblank = pixelX * Hblank;
		if (isNearEqual(Frag_UV.x, hblank-pixelX, pixelX)) {
			if (mod(floor(gl_FragCoord.y), 4) < 2.0) {
				Out_Color.r = 1.0;
				Out_Color.g = 1.0;
				Out_Color.b = 1.0;
				Out_Color.a = 0.1;
				return;
			}
		}

		// frame flyback guide
		float lastNewFrameAtScanline = pixelY * LastNewFrameAtScanline;
		if (isNearEqual(Frag_UV.y, lastNewFrameAtScanline-pixelY, pixelY)) {
			if (mod(floor(gl_FragCoord.x), 8) < 3.0) {
				Out_Color.r = 1.0;
				Out_Color.g = 0.0;
				Out_Color.b = 1.0;
				Out_Color.a = 0.1;
				return;
			}
		}
	}

	// when ShowCursor is true then there is some additional processing we need to perform
	if (ShowCursor == 1) {
		// draw cursor if pixel is at the last x/y position
		if (lastY >= 0 && lastX >= 0) {
			if (isNearEqual(Frag_UV.y, lastY+texelY, cursorSize*texelY) && isNearEqual(Frag_UV.x, lastX+texelX, cursorSize*texelX/2)) {
				Out_Color.r = 1.0;
				Out_Color.g = 1.0;
				Out_Color.b = 1.0;
				Out_Color.a = 1.0;
				return;
			}
		}

		// draw off-screen cursor for HBLANK
		if (lastX < 0 && isNearEqual(Frag_UV.y, lastY+texelY, cursorSize*texelY) && isNearEqual(Frag_UV.x, 0, cursorSize*texelX/2)) {
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
			if (isNearEqual(Frag_UV.x, lastX, cursorSize * texelX/2)) {
				// top of screen
				if (lastY < 0 && isNearEqual(Frag_UV.y, 0, cursorSize*texelY)) {
					Out_Color.r = 1.0;
					Out_Color.g = 0.0;
					Out_Color.b = 0.0;
					Out_Color.a = 1.0;
					return;
				}
			
				// bottom of screen (knocking a pixel off the scanline
				// boundary check to make sure the cursor is visible)
				if (lastY > visibleBottom-pixelY && isNearEqual(Frag_UV.y, visibleBottom, cursorSize*texelY)) {
					Out_Color.r = 1.0;
					Out_Color.g = 0.0;
					Out_Color.b = 0.0;
					Out_Color.a = 1.0;
					return;
				}
			}

			// when HBLANK and VBLANK are both active
			if (lastX < 0 && isNearEqual(Frag_UV.x, 0, cursorSize*texelX/2)) {
				// top/left corner of screen
				if (lastY < 0 && isNearEqual(Frag_UV.y, 0, cursorSize*texelY)) {
					Out_Color.r = 1.0;
					Out_Color.g = 0.0;
					Out_Color.b = 0.0;
					Out_Color.a = 1.0;
					return;
				}

				// bottom/left corner of screen (knocking a pixel off the
				// scanline boundary check to make sure the cursor is
				// visible)
				if (lastY > visibleBottom-pixelY && isNearEqual(Frag_UV.y, visibleBottom, cursorSize*texelY)) {
					Out_Color.r = 1.0;
					Out_Color.g = 0.0;
					Out_Color.b = 0.0;
					Out_Color.a = 1.0;
					return;
				}
			}
		}

		// painting effect draws pixels with faded alpha if lastX and lastY
		// are less than rendering Frag_UV.
		//
		// as a special case, we ignore the first scanline and do not fade the
		// previous image on a brand new frame. note that we're using the
		// unadjusted LastY value for this
		if (LastY > 0) {
			if (Frag_UV.y > lastY+texelY || (isNearEqual(Frag_UV.y, lastY+texelY, texelY) && Frag_UV.x > lastX+texelX)) {
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

