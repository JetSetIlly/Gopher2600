#version 150

uniform sampler2D Texture;
uniform vec2 ScreenDim;
in vec2 Frag_UV;
in vec4 Frag_Color;
out vec4 Out_Color;

void main()
{
	float texelX = 2.0/ScreenDim.x;
	float texelY = 2.0/ScreenDim.y;
    vec4 tl = texture(Texture, Frag_UV);
    vec4 tr = texture(Texture, Frag_UV + vec2(texelX, 0.0));
    vec4 bl = texture(Texture, Frag_UV + vec2(0.0, texelY));
    vec4 br = texture(Texture, Frag_UV + vec2(texelX, texelY));
    vec2 f = fract(Frag_UV * ScreenDim);
    vec4 tA = mix(tl, tr, f.x);
    vec4 tB = mix(bl, br, f.x);
    Out_Color = mix(tA, tB, f.y);
}
