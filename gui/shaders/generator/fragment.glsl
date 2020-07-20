// bending and colour splitting in fragment shader cribbed from shadertoy
// project: https://www.shadertoy.com/view/4sf3Dr

uniform int ImageType;
uniform int PixelPerfect;
uniform int ShowScreenDraw; // false <= 0; true > 0
uniform int Cropped; // false <= 0; true > 0
uniform vec2 ScreenDim;
uniform vec2 CropScreenDim;
uniform float ScalingX;
uniform float ScalingY;
uniform float LastX; 
uniform float LastY;
uniform float Hblank;
uniform float TopScanline;
uniform float BotScanline;
uniform float AnimTime;

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
	// imgui texture
	if (ImageType == 0) {
		Out_Color = vec4(Frag_Color.rgb, Frag_Color.a * texture(Texture, Frag_UV.st).r);
		return;
	}

	// if this is the overlay texture then we're done
	if (ImageType == 2) {
		Out_Color = Frag_Color * texture(Texture, Frag_UV.st);
		return;
	}

	vec2 coords = Frag_UV.xy;

	// bring geometry values into workable range
	float hblank;
	float topScanline;
	float botScanline;
	float lastX;
	float lastY;

	// the size of one texel (used for painting and cursor positioning)
	float texelX;
	float texelY;

	// debug tv screen texture
	if (ImageType == 1) {

		if (Cropped > 0) {
			texelX = ScalingX / CropScreenDim.x;
			texelY = ScalingY / CropScreenDim.y;
			hblank = Hblank / CropScreenDim.x;
			lastX = LastX / CropScreenDim.x;
			topScanline = 0;
			botScanline = (BotScanline - TopScanline) / CropScreenDim.y;

			// the LastY coordinate refers to the full-frame scanline. the cropped
			// texture however counts from zero at the visible edge so we need to
			// adjust the lastY value by the TopScanline value.
			//
			// note that there's no need to do this for LastX because the
			// horizontal position is counted from -68 in all instances.
			lastY = (LastY - TopScanline) / CropScreenDim.y;
		} else {
			texelX = ScalingX / ScreenDim.x;
			texelY = ScalingY / ScreenDim.y;
			hblank = Hblank / ScreenDim.x;
			topScanline = TopScanline / ScreenDim.y;
			botScanline = BotScanline / ScreenDim.y;
			lastX = LastX / ScreenDim.x;
			lastY = LastY / ScreenDim.y;
		}

		// pixels are texels without the scaling applied
		float pixelX = texelX / ScalingX;
		float pixelY = texelY / ScalingY;

		// if the entire frame is being shown then plot the screen guides
		if (Cropped < 0) {
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

		// when ShowScreenDraw is true then there's some additional image
		// processing we need to perform:
		//	- fade anything left over from previous frame
		//	- draw cursor indicator
		if (ShowScreenDraw > 0) {
			
			// draw cursor if pixel is at the last x/y position
			if (lastY >= 0 && lastX >= 0) {
				if (isNearEqual(coords.y, lastY+texelY, cursorSize*texelY) && isNearEqual(coords.x, lastX+texelX, cursorSize*texelX/2)) {
					Out_Color.r = 1.0;
					Out_Color.g = 1.0;
					Out_Color.b = 1.0;
					Out_Color.a = AnimTime;
					return;
				}
			}

			// draw off-screen cursor for HBLANK
			if (lastX < 0 && isNearEqual(coords.y, lastY+texelY, cursorSize*texelY) && isNearEqual(coords.x, 0, cursorSize*texelX/2)) {
				Out_Color.r = 1.0;
				Out_Color.a = AnimTime;
				return;
			}

			// for cropped screens there are a few more conditions that we need to
			// consider for drawing an off-screen cursor
			if (Cropped > 0) {

				// when VBLANK is active but HBLANK is off
				if (isNearEqual(coords.x, lastX, cursorSize * texelX/2)) {
					// top of screen
					if (lastY < 0 && isNearEqual(coords.y, 0, cursorSize*texelY)) {
						Out_Color.r = 1.0;
						Out_Color.a = AnimTime;
						return;
					}
				
					// bottom of screen (knocking a pixel off the scanline
					// boundary check to make sure the cursor is visible)
					if (lastY > botScanline-pixelY && isNearEqual(coords.y, botScanline, cursorSize*texelY)) {
						Out_Color.r = 1.0;
						Out_Color.a = AnimTime;
						return;
					}
				}

				// when HBLANK and VBLANK are both active
				if (lastX < 0 && isNearEqual(coords.x, 0, cursorSize*texelX/2)) {
					// top/left corner of screen
					if (lastY < 0 && isNearEqual(coords.y, 0, cursorSize*texelY)) {
						Out_Color.r = 1.0;
						Out_Color.a = AnimTime;
						return;
					}

					// bottom/left corner of screen (knocking a pixel off the
					// scanline boundary check to make sure the cursor is
					// visible)
					if (lastY > botScanline-pixelY && isNearEqual(coords.y, botScanline, cursorSize*texelY)) {
						Out_Color.r = 1.0;
						Out_Color.a = AnimTime;
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
	} else {
		texelX = ScalingX / CropScreenDim.x;
		texelY = ScalingY / CropScreenDim.y;
	}

	// if pixel-perfect	rendering is selected then there's nothing much more to do
	if (PixelPerfect == 1) {
		Out_Color = Frag_Color * texture(Texture, Frag_UV.st);
		return;
	}

	// only apply CRT effects on the "cropped" area of the screen. we can think
	// of the cropped area as the "play" area
	if (Cropped < 0 && (coords.x < hblank || coords.y < topScanline || coords.y > botScanline)) {
		Out_Color = Frag_Color * texture(Texture, Frag_UV.st);
		return;
	}

	// basic CRT effects

	// split color channels
	vec2 split;
	split.x = 0.0001;
	split.y = 0.001;
	if (coords.x > split.x && coords.y > split.y) {
		Out_Color.r = texture(Texture, vec2(coords.x-split.x, coords.y)).r;
		Out_Color.g = texture(Texture, vec2(coords.x, coords.y-split.y)).g;
		Out_Color.b = texture(Texture, vec2(coords.x+split.x, coords.y)).b;
		Out_Color.a = Frag_Color.a;
	}

	// vignette effect
	float vignette;
	if (Cropped > 0) {
		vignette = (10.0*coords.x*coords.y*(1.0-coords.x)*(1.0-coords.y));
	} else {
		// f is used to factor the vignette value. In the "cropped" branch we
		// use a factor value of 10. to visually mimic the vignette effect a
		// value of about 25 is required (using Pitfall as a template). I don't
		// understand this well enough to say for sure what the relationship
		// between 25 and 10 is, but the following ratio between
		// cropped/uncropped widths gives us a value of 23.5
		float f =ScreenDim.x/(ScreenDim.x-CropScreenDim.x);
		vignette = (f*(coords.x-hblank)*(coords.y-topScanline)*(1.0-coords.x)*(1.0-coords.y));
	}
	Out_Color.rgb *= pow(vignette, 0.10) * 1.2;

	// scanline effect
	float oneLine = gl_FragCoord.y/gl_FragCoord.y;
	if ( isNearEqual(mod(gl_FragCoord.y, 3.0*oneLine), 0.0, oneLine) ) {
		Out_Color.a = Frag_Color.a * 0.85;
	}

	// bend screen
	/* if (ImageType == 3) { */
	/* 	float xbend = 6.0; */
	/* 	float ybend = 5.0; */
	/* 	coords = (coords - 0.5) * 1.85; */
	/* 	coords *= 1.11; */	
	/* 	coords.x *= 1.0 + pow((abs(coords.y) / xbend), 2.0); */
	/* 	coords.y *= 1.0 + pow((abs(coords.x) / ybend), 2.0); */
	/* 	coords  = (coords / 2.05) + 0.5; */

	/* 	// crop tiling */
	/* 	if (coords.x < 0.0 || coords.x > 1.0 || coords.y < 0.0 || coords.y > 1.0 ) { */
	/* 		discard; */
	/* 	} */
	/* } */
}
