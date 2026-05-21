uniform sampler2D Texture;
in vec2 Frag_UV;
in vec4 Frag_Color;
out vec4 Out_Color;

uniform int IsCropped; 
uniform int ShowCursor;  
uniform vec2 ScreenDim;
uniform float ScalingX;
uniform float ScalingY;
uniform float LastX; 
uniform float LastY;
uniform float Hblank;
uniform float TotalScanlines;
uniform float TopScanline;

// the top and bottom scanlines to show. in the case of IsCropped then these
// values will be used to draw the screen guides
uniform float VisibleTop;
uniform float VisibleBottom;

// zoom and pivot control the amount of magnification
uniform float Zoom;
uniform vec2 Pivot;

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

// transformed uv after zoom
vec2 uv;

void prepareDbgScr() {
	pixelX = 1.0 / ScreenDim.x;
	pixelY = 1.0 / ScreenDim.y;
	texelX = pixelX * ScalingX;
	texelY = pixelY * ScalingY;
	lastX = pixelX * LastX;
	lastY = pixelY * LastY;
	uv = Pivot + (Frag_UV - Pivot) / Zoom;
}

bool isNearEqual(float x, float y, float epsilon)
{
	return abs(x - y) <= epsilon;
}
