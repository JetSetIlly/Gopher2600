#version 150

uniform float ScaledWidth;
uniform float ScaledHeight;
uniform float UnscaledWidth;
uniform float UnscaledHeight;
uniform sampler2D Texture;
uniform sampler2D UnscaledTexture;
in vec2 Frag_UV;
in vec4 Frag_Color;
out vec4 Out_Color;

void main()
{
	Out_Color = Frag_Color * texture(Texture, Frag_UV.st);
    Out_Color = mix(Out_Color, texture(UnscaledTexture, Frag_UV.st), 1.0);

    Out_Color = mix(Out_Color, texture(UnscaledTexture, Frag_UV.st-vec2(0.0, 0.00025)), 0.75);
    Out_Color = mix(Out_Color, texture(UnscaledTexture, Frag_UV.st-vec2(0.0, 0.0005)), 0.75);
    Out_Color = mix(Out_Color, texture(UnscaledTexture, Frag_UV.st-vec2(0.0, 0.00075)), 0.75);
    Out_Color = mix(Out_Color, texture(UnscaledTexture, Frag_UV.st-vec2(0.0, 0.001)), 0.75);
    Out_Color = mix(Out_Color, texture(UnscaledTexture, Frag_UV.st-vec2(0.0, 0.00125)), 0.5);
    Out_Color = mix(Out_Color, texture(UnscaledTexture, Frag_UV.st-vec2(0.0, 0.0015)), 0.5);
    Out_Color = mix(Out_Color, texture(UnscaledTexture, Frag_UV.st-vec2(0.0, 0.00175)), 0.5);
    Out_Color = mix(Out_Color, texture(UnscaledTexture, Frag_UV.st-vec2(0.0, 0.002)), 0.5);
    Out_Color = mix(Out_Color, texture(UnscaledTexture, Frag_UV.st-vec2(0.0, 0.00225)), 0.25);
    Out_Color = mix(Out_Color, texture(UnscaledTexture, Frag_UV.st-vec2(0.0, 0.0025)), 0.25);
    Out_Color = mix(Out_Color, texture(UnscaledTexture, Frag_UV.st-vec2(0.0, 0.00275)), 0.25);
    Out_Color = mix(Out_Color, texture(UnscaledTexture, Frag_UV.st-vec2(0.0, 0.003)), 0.25);

    Out_Color = mix(Out_Color, texture(UnscaledTexture, Frag_UV.st+vec2(0.0, 0.00025)), 0.75);
    Out_Color = mix(Out_Color, texture(UnscaledTexture, Frag_UV.st+vec2(0.0, 0.0005)), 0.75);
    Out_Color = mix(Out_Color, texture(UnscaledTexture, Frag_UV.st+vec2(0.0, 0.00075)), 0.75);
    Out_Color = mix(Out_Color, texture(UnscaledTexture, Frag_UV.st+vec2(0.0, 0.001)), 0.75);
    Out_Color = mix(Out_Color, texture(UnscaledTexture, Frag_UV.st+vec2(0.0, 0.00125)), 0.5);
    Out_Color = mix(Out_Color, texture(UnscaledTexture, Frag_UV.st+vec2(0.0, 0.0015)), 0.5);
    Out_Color = mix(Out_Color, texture(UnscaledTexture, Frag_UV.st+vec2(0.0, 0.00175)), 0.5);
    Out_Color = mix(Out_Color, texture(UnscaledTexture, Frag_UV.st+vec2(0.0, 0.002)), 0.5);
    Out_Color = mix(Out_Color, texture(UnscaledTexture, Frag_UV.st+vec2(0.0, 0.00225)), 0.25);
    Out_Color = mix(Out_Color, texture(UnscaledTexture, Frag_UV.st+vec2(0.0, 0.0025)), 0.25);
    Out_Color = mix(Out_Color, texture(UnscaledTexture, Frag_UV.st+vec2(0.0, 0.00275)), 0.25);
    Out_Color = mix(Out_Color, texture(UnscaledTexture, Frag_UV.st+vec2(0.0, 0.003)), 0.25);
}
