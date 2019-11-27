package symbols_test

import (
	"gopher2600/symbols"
	"testing"
)

const expectedSymbolsList = `Locations
---------

Read Symbols
-----------
0x0000 -> CXM0P
0x0001 -> CXM1P
0x0002 -> CXP0FB
0x0003 -> CXP1FB
0x0004 -> CXM0FB
0x0005 -> CXM1FB
0x0006 -> CXBLPF
0x0007 -> CXPPMM
0x0008 -> INPT0
0x0009 -> INPT1
0x000a -> INPT2
0x000b -> INPT3
0x000c -> INPT4
0x000d -> INPT5
0x000e -> SWCHA
0x000f -> SWACNT
0x0010 -> SWCHB
0x0011 -> SWBCNT
0x0012 -> INTIM
0x0013 -> TIMINT

Write Symbols
------------
0x0000 -> VSYNC
0x0001 -> VBLANK
0x0002 -> WSYNC
0x0003 -> RSYNC
0x0004 -> NUSIZ0
0x0005 -> NUSIZ1
0x0006 -> COLUP0
0x0007 -> COLUP1
0x0008 -> COLUPF
0x0009 -> COLUBK
0x000a -> CTRLPF
0x000b -> REFP0
0x000c -> REFP1
0x000d -> PF0
0x000e -> PF1
0x000f -> PF2
0x0010 -> RESP0
0x0011 -> RESP1
0x0012 -> RESM0
0x0013 -> RESM1
0x0014 -> RESBL
0x0015 -> AUDC0
0x0016 -> AUDC1
0x0017 -> AUDF0
0x0018 -> AUDF1
0x0019 -> AUDV0
0x001a -> AUDV1
0x001b -> GRP0
0x001c -> GRP1
0x001d -> ENAM0
0x001e -> ENAM1
0x001f -> ENABL
0x0020 -> HMP0
0x0021 -> HMP1
0x0022 -> HMM0
0x0023 -> HMM1
0x0024 -> HMBL
0x0025 -> VDELP0
0x0026 -> VDELP1
0x0027 -> VDELBL
0x0028 -> RESMP0
0x0029 -> RESMP1
0x002a -> HMOVE
0x002b -> HMCLR
0x002c -> CXCLR
0x002d -> SWCHA
0x002e -> SWACNT
0x002f -> TIM1T
0x0030 -> TIM8T
0x0031 -> TIM64T
0x0032 -> TIM1024
`

type testWriter struct {
	buffer []byte
}

func (tw *testWriter) Write(p []byte) (n int, err error) {
	tw.buffer = append(tw.buffer, p...)
	return len(p), nil
}

func (tw *testWriter) cmp(s string) bool {
	return s == string(tw.buffer)
}

func TestDefaultSymbols(t *testing.T) {
	syms, err := symbols.ReadSymbolsFile("")
	if err != nil {
		t.Errorf("unexpected error (%s)", err)
	}

	tw := &testWriter{}

	syms.ListSymbols(tw)

	if !tw.cmp(expectedSymbolsList) {
		t.Errorf("default symbols list is wrong")
	}
}
