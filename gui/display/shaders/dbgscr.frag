uniform sampler2D Texture;
in vec2 Frag_UV;
in vec4 Frag_Color;
out vec4 Out_Color;

void main()
{
	prepareDbgScr();
	Out_Color = Frag_Color * texture(Texture, Frag_UV);

	if (IsCropped == 1) {
		visibleBottom = (VisibleBottom - VisibleTop) / ScreenDim.y;
		lastY -=  pixelY * VisibleTop;
	} else {
		// visible screen guides
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

		// hblank guide
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

		// frame flyback guide
		float lastNewFrameAtScanline = pixelY * LastNewFrameAtScanline;
		if (isNearEqual(Frag_UV.y, lastNewFrameAtScanline, pixelY)) {
			if (mod(floor(gl_FragCoord.x), 8) < 3.0) {
				Out_Color.r = 1.0;
				Out_Color.g = 0.0;
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

		// alternative magnification guide (dotted line around area)
		/*
		if (Frag_UV.x > xmin && Frag_UV.x < xmax && (isNearEqual(Frag_UV.y, ymin, pixelY) || isNearEqual(Frag_UV.y, ymax, pixelY)) ) {
			if (mod(floor(gl_FragCoord.x), 8) < 5.0) {
				Out_Color.r = 1.0;
				Out_Color.g = 1.0;
				Out_Color.b = 1.0;
				Out_Color.a = 1.0;
			}
		}
		if (Frag_UV.y > ymin && Frag_UV.y < ymax && (isNearEqual(Frag_UV.x, xmin, pixelX) || isNearEqual(Frag_UV.x, xmax, pixelX))) {
			if (mod(floor(gl_FragCoord.y), 8) < 5.0) {
				Out_Color.r = 1.0;
				Out_Color.g = 1.0;
				Out_Color.b = 1.0;
				Out_Color.a = 1.0;
			}
		}
		*/
	}

	// show cursor. the cursor illustrates the *most recent* pixel to be drawn
	if (ShowCursor == 1) {
		// draw cursor if pixel is at the last x/y position
		if (lastY >= 0 && lastX >= 0) {
			if (isNearEqual(Frag_UV.y, lastY+texelY, texelY) && isNearEqual(Frag_UV.x, lastX+texelX/2, texelX/2)) {
				Out_Color.r = 1.0;
				Out_Color.g = 1.0;
				Out_Color.b = 1.0;
				Out_Color.a = 1.0;
				return;
			}
		}

		// draw off-screen cursor for HBLANK
		if (lastX < 0 && isNearEqual(Frag_UV.y, lastY+texelY, texelY) && isNearEqual(Frag_UV.x, 0, texelX/2)) {
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
			if (isNearEqual(Frag_UV.x, lastX, texelX/2)) {
				// top of screen
				if (lastY < 0 && isNearEqual(Frag_UV.y, 0, texelY)) {
					Out_Color.r = 1.0;
					Out_Color.g = 0.0;
					Out_Color.b = 0.0;
					Out_Color.a = 1.0;
					return;
				}
			
				// bottom of screen (knocking a pixel off the scanline
				// boundary check to make sure the cursor is visible)
				if (lastY > visibleBottom-pixelY && isNearEqual(Frag_UV.y, visibleBottom, texelY)) {
					Out_Color.r = 1.0;
					Out_Color.g = 0.0;
					Out_Color.b = 0.0;
					Out_Color.a = 1.0;
					return;
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
					return;
				}

				// bottom/left corner of screen (knocking a pixel off the
				// scanline boundary check to make sure the cursor is
				// visible)
				if (lastY > visibleBottom-pixelY && isNearEqual(Frag_UV.y, visibleBottom, texelY)) {
					Out_Color.r = 1.0;
					Out_Color.g = 0.0;
					Out_Color.b = 0.0;
					Out_Color.a = 1.0;
					return;
				}
			}
		}

		// painting effect but if the emulation is still on the first line of
		// the TV frame
		if (LastY > 0) {
			Out_Color = paintingEffect(Frag_UV, Out_Color);
		}
	}
}

