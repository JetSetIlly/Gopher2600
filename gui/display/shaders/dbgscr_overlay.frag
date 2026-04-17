void main()
{
	prepareDbgScr();
	Out_Color = Frag_Color * texture(Texture, uv);
	if (Out_Color.a > 0.0) {
		Out_Color.a = 0.3;
	}
}
