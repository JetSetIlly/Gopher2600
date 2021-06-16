#version 150

uniform sampler2D Texture;
uniform vec2 ScreenDim;
in vec2 Frag_UV;
in vec4 Frag_Color;
out vec4 Out_Color;

void main()
{
  float texelX = 1.0 / ScreenDim.x;
  float texelY = 1.0 / ScreenDim.y;
  vec4 sum;
  sum += -1. * texture2D(Texture, Frag_UV + vec2( -1.0 * texelX , 0.0 * texelY));
  sum += -1. * texture2D(Texture, Frag_UV + vec2( 0.0 * texelX , -1.0 * texelY));
  sum += 5. * texture2D(Texture, Frag_UV + vec2( 0.0 * texelX , 0.0 * texelY));
  sum += -1. * texture2D(Texture, Frag_UV + vec2( 0.0 * texelX , 1.0 * texelY));
  sum += -1. * texture2D(Texture, Frag_UV + vec2( 1.0 * texelX , 0.0 * texelY));
  Out_Color = sum;
}
