uniform int IsCropped; 
uniform int ShowCursor;  
uniform vec2 ScreenDim;
uniform float ScalingX;
uniform float ScalingY;
uniform float LastX; 
uniform float LastY;
uniform float Hblank;
uniform float LastNewFrameAtScanline;

// the top abd bottom scanlines to show. in the case of IsCropped then these
// values will be used to draw the screen guides
uniform float VisibleTop;
uniform float VisibleBottom;

uniform int MagShow;
uniform float MagXmin; 
uniform float MagXmax; 
uniform float MagYmin; 
uniform float MagYmax; 

// value of one pixel
float pixelX;
float pixelY;

// the size of one texel (used for painting and cursor positioning)
float texelX;
float texelY;

// adjusted last x/y coordinates. lastY depends on IsCropped
float lastX;
float lastY;

// bottom screen boundary. depends on IsCropped
float visibleBottom;

void prepareDbgScr() {
	pixelX = 1.0 / ScreenDim.x;
	pixelY = 1.0 / ScreenDim.y;
	texelX = pixelX * ScalingX;
	texelY = pixelY * ScalingY;
	lastX = pixelX * LastX;
	lastY = pixelY * LastY;
}

bool isNearEqual(float x, float y, float epsilon)
{
	return abs(x - y) <= epsilon;
}

vec4 paintingEffect(vec2 uv, vec4 col) {
	if (uv.y > lastY+texelY || (isNearEqual(uv.y, lastY+texelY, texelY) && uv.x > lastX+texelX)) {
		col = mix(col, vec4(col.rgb, 0.0), 0.5);
	}
	return col;
}
