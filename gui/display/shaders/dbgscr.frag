uniform sampler2D Texture;
in vec2 Frag_UV;
in vec4 Frag_Color;
out vec4 Out_Color;

bool cursor();

void main()
{
	prepareDbgScr();
	Out_Color = Frag_Color * texture(Texture, Frag_UV);

	if (IsCropped == 1) {
		visibleBottom = (VisibleBottom - VisibleTop) / ScreenDim.y;
		lastY -=  pixelY * VisibleTop;
	}

	// draw cursor as a priority. it is always drawn in preference to any other pixel
	if (ShowCursor == 1) {
		if (cursor() == true) {
			return;
		}
	}

	if (IsCropped == 0) {
		float visibleTop = pixelY * VisibleTop;
		visibleBottom = pixelY * VisibleBottom;

		// visibleBottom is adjusted by texelY for the test because we want the
		// guide to show on the outer edge of the visible boundary
		if (isNearEqual(Frag_UV.y, visibleTop, pixelY) || isNearEqual(Frag_UV.y, visibleBottom+texelY, pixelY)) {
			if (mod(floor(gl_FragCoord.x), 4) < 2.0) {
				Out_Color.r = 1.0;
				Out_Color.g = 1.0;
				Out_Color.b = 1.0;
				Out_Color.a = 0.1;
				return;
			}
		}

		// no signal area extends from the total scanlines pixel to the
		// bottom of the texture
		float totalScanlines = pixelY * TotalScanlines;
		float topScanline = pixelY * TopScanline;
		if (Frag_UV.y >= totalScanlines || Frag_UV.y < topScanline) {
			// adding x and y frag coords creates a diagonal stripe
			if (mod(floor(gl_FragCoord.x+gl_FragCoord.y), 8) < 3.0) {
				Out_Color.r = 0.05;
				Out_Color.g = 0.05;
				Out_Color.b = 0.05;
				Out_Color.a = 0.8;
				return;
			} else {
				Out_Color.r = 0.03;
				Out_Color.g = 0.03;
				Out_Color.b = 0.03;
				Out_Color.a = 0.8;
				return;
			}
		}

		// hblank guide. doesn't extend into the no signal area
		float hblank = pixelX * Hblank;
		if (isNearEqual(Frag_UV.x, hblank, pixelX)) {
			if (mod(floor(gl_FragCoord.y), 4) < 2.0) {
				Out_Color.r = 1.0;
				Out_Color.g = 1.0;
				Out_Color.b = 1.0;
				Out_Color.a = 0.1;
				return;
			}
		}
	}

	// magnification guide
	if (MagShow == 1) {
		float xmin = pixelX * MagXmin;
		float xmax = pixelX * MagXmax;
		float ymin = pixelY * MagYmin;
		float ymax = pixelY * MagYmax;

		// fade out magnified area
		if (Frag_UV.x > xmin && Frag_UV.x < xmax && Frag_UV.y > ymin && Frag_UV.y < ymax) {
		 		Out_Color.r += 0.1;
		 		Out_Color.g += 0.1;
		 		Out_Color.b += 0.1;
				Out_Color.a = 0.5;
		}

		// pixel thickness of dotted outline
		#define pixelThickness 0.0

		if (pixelThickness > 0.0) {
			// dotted line around magnification area
			if (Frag_UV.x > xmin && Frag_UV.x < xmax && (isNearEqual(Frag_UV.y, ymin, pixelY*pixelThickness) || isNearEqual(Frag_UV.y, ymax, pixelY*pixelThickness)) ) {
				if (mod(floor(gl_FragCoord.x), 8) < 5.0) {
					Out_Color.r = 0.8;
					Out_Color.g = 0.8;
					Out_Color.b = 0.8;
					Out_Color.a = 1.0;
				}
			}
			if (Frag_UV.y > ymin && Frag_UV.y < ymax && (isNearEqual(Frag_UV.x, xmin, pixelX*pixelThickness) || isNearEqual(Frag_UV.x, xmax, pixelX*pixelThickness))) {
				if (mod(floor(gl_FragCoord.y), 8) < 5.0) {
					Out_Color.r = 0.8;
					Out_Color.g = 0.8;
					Out_Color.b = 0.8;
					Out_Color.a = 1.0;
				}
			}
		}
	}


	// painting effect but if the emulation is still on the first line of
	// the TV frame
	if (ShowCursor == 1) {
		if (LastY > 0) {
			Out_Color = paintingEffect(Frag_UV, Out_Color);
		}
	}
}

bool cursor() {
	// draw cursor if pixel is at the last x/y position
	if (lastY >= 0 && lastX >= 0) {
		if (isNearEqual(Frag_UV.y, lastY+texelY, texelY) && isNearEqual(Frag_UV.x, lastX+texelX/2, texelX/2)) {
			Out_Color.r = 1.0;
			Out_Color.g = 1.0;
			Out_Color.b = 1.0;
			Out_Color.a = 1.0;
			return true;
		}
	}

	// draw off-screen cursor for HBLANK
	if (lastX < 0 && isNearEqual(Frag_UV.y, lastY+texelY, texelY) && isNearEqual(Frag_UV.x, 0, texelX/2)) {
		Out_Color.r = 1.0;
		Out_Color.g = 0.0;
		Out_Color.b = 0.0;
		Out_Color.a = 1.0;
		return true;
	}

	// for cropped screens there are a few more conditions that we need to
	// consider for drawing an off-screen cursor
	if (IsCropped == 1) {
		// when VBLANK is active but HBLANK is off
		if (isNearEqual(Frag_UV.x, lastX, texelX/2)) {
			// top of screen
			if (lastY < 0 && isNearEqual(Frag_UV.y, 0, texelY)) {
				Out_Color.r = 1.0;
				Out_Color.g = 0.0;
				Out_Color.b = 0.0;
				Out_Color.a = 1.0;
				return true;
			}
		
			// bottom of screen (knocking a pixel off the scanline
			// boundary check to make sure the cursor is visible)
			if (lastY > visibleBottom-pixelY && isNearEqual(Frag_UV.y, visibleBottom, texelY)) {
				Out_Color.r = 1.0;
				Out_Color.g = 0.0;
				Out_Color.b = 0.0;
				Out_Color.a = 1.0;
				return true;
			}
		}

		// when HBLANK and VBLANK are both active
		if (lastX < 0 && isNearEqual(Frag_UV.x, 0, texelX/2)) {
			// top/left corner of screen
			if (lastY < 0 && isNearEqual(Frag_UV.y, 0, texelY)) {
				Out_Color.r = 1.0;
				Out_Color.g = 0.0;
				Out_Color.b = 0.0;
				Out_Color.a = 1.0;
				return true;
			}

			// bottom/left corner of screen (knocking a pixel off the
			// scanline boundary check to make sure the cursor is
			// visible)
			if (lastY > visibleBottom-pixelY && isNearEqual(Frag_UV.y, visibleBottom, texelY)) {
				Out_Color.r = 1.0;
				Out_Color.g = 0.0;
				Out_Color.b = 0.0;
				Out_Color.a = 1.0;
				return true;
			}
		}
	}

	return false;
}
