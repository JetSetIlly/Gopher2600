uniform sampler2D Texture;
in vec2 Frag_UV;
in vec4 Frag_Color;
out vec4 Out_Color;

// sharpen function taken from (licenced under CC0):
// https://gist.github.com/Beefster09/7264303ee4b4b2086f372f1e70e8eddd

#define sharpness 4

float sharpen(float pix_coord) {
    float norm = (fract(pix_coord) - 0.5) * 2.0;
    float norm2 = norm * norm;
    return floor(pix_coord) + norm * pow(norm2, sharpness) / 2.0 + 0.5;
}

void main() {
    vec2 vres = textureSize(Texture, 0);
    Out_Color = texture(Texture, vec2(
        sharpen(Frag_UV.x * vres.x) / vres.x,
        sharpen(Frag_UV.y * vres.y) / vres.y
    ));
}
