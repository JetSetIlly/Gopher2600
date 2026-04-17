bool cursor();

void main()
{
	prepareDbgScr();
	Out_Color = Frag_Color * texture(Texture, uv);

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
		if (isNearEqual(uv.y, visibleTop, pixelY) || isNearEqual(uv.y, visibleBottom+texelY, pixelY)) {
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
		if (uv.y >= totalScanlines || uv.y < topScanline) {
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
		if (isNearEqual(uv.x, hblank, pixelX)) {
			if (mod(floor(gl_FragCoord.y), 4) < 2.0) {
				Out_Color.r = 1.0;
				Out_Color.g = 1.0;
				Out_Color.b = 1.0;
				Out_Color.a = 0.1;
				return;
			}
		}
	}

	// painting effect but if the emulation is still on the first line of the TV frame
	if (ShowCursor == 1) {
		if ((LastY > 0) || (LastY== 0 && LastX >= 3)) {
			if (uv.y > lastY+texelY || (isNearEqual(uv.y, lastY+texelY, texelY) && uv.x > lastX+texelX)) {
				Out_Color = mix(Out_Color, vec4(Out_Color.rgb, 0.0), 0.5);
			}
		}
	}
}

bool cursor() {
	// draw cursor if pixel is at the last x/y position
	if (lastY >= 0 && lastX >= 0) {
		if (isNearEqual(uv.y, lastY+texelY, texelY) && isNearEqual(uv.x, lastX+texelX/2, texelX/2)) {
			Out_Color.r = 1.0;
			Out_Color.g = 1.0;
			Out_Color.b = 1.0;
			Out_Color.a = 1.0;
			return true;
		}
	}

	// draw off-screen cursor for HBLANK
	if (lastX < 0 && isNearEqual(uv.y, lastY+texelY, texelY) && isNearEqual(uv.x, 0, texelX/2)) {
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
		if (isNearEqual(uv.x, lastX, texelX/2)) {
			// top of screen
			if (lastY < 0 && isNearEqual(uv.y, 0, texelY)) {
				Out_Color.r = 1.0;
				Out_Color.g = 0.0;
				Out_Color.b = 0.0;
				Out_Color.a = 1.0;
				return true;
			}
		
			// bottom of screen (knocking a pixel off the scanline
			// boundary check to make sure the cursor is visible)
			if (lastY > visibleBottom-pixelY && isNearEqual(uv.y, visibleBottom, texelY)) {
				Out_Color.r = 1.0;
				Out_Color.g = 0.0;
				Out_Color.b = 0.0;
				Out_Color.a = 1.0;
				return true;
			}
		}

		// when HBLANK and VBLANK are both active
		if (lastX < 0 && isNearEqual(uv.x, 0, texelX/2)) {
			// top/left corner of screen
			if (lastY < 0 && isNearEqual(uv.y, 0, texelY)) {
				Out_Color.r = 1.0;
				Out_Color.g = 0.0;
				Out_Color.b = 0.0;
				Out_Color.a = 1.0;
				return true;
			}

			// bottom/left corner of screen (knocking a pixel off the
			// scanline boundary check to make sure the cursor is
			// visible)
			if (lastY > visibleBottom-pixelY && isNearEqual(uv.y, visibleBottom, texelY)) {
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
