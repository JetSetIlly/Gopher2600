// This file is part of Gopher2600.
//
// Gopher2600 is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// Gopher2600 is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with Gopher2600.  If not, see <https://www.gnu.org/licenses/>.

package elf

func getStrongArmDefinition(mem *elfMemory, name string) (bool, uint32, error) {
	var tgt uint32
	var err error

	switch name {
	case "vcsCopyOverblankToRiotRam":
		tgt, err = mem.relocateStrongArmFunction(strongArmFunctionSpec{
			name:     name,
			function: vcsCopyOverblankToRiotRam,
			support:  false,
		})
	case "vcsLibInit":
		tgt, err = mem.relocateStrongArmFunction(strongArmFunctionSpec{
			name:     name,
			function: vcsLibInit,
			support:  false,
		})
	case "vcsInitBusStuffing":
		tgt, err = mem.relocateStrongArmFunction(strongArmFunctionSpec{
			name:     name,
			function: vcsInitBusStuffing,
			support:  true,
		})
	case "updateLookupTables":
		tgt, err = mem.relocateStrongArmFunction(strongArmFunctionSpec{
			name: name,
			function: func(mem *elfMemory) {
				mem.strongarm.updateLookupTables()
			},
			support: true,
		})
	case "vcsWrite3":
		mem.usesBusStuffing = true
		tgt, err = mem.relocateStrongArmFunction(strongArmFunctionSpec{
			name:     name,
			function: vcsWrite3,
			support:  false,
		})
	case "vcsJmp3":
		tgt, err = mem.relocateStrongArmFunction(strongArmFunctionSpec{
			name:     name,
			function: vcsJmp3,
			support:  false,
		})
	case "vcsLda2":
		tgt, err = mem.relocateStrongArmFunction(strongArmFunctionSpec{
			name:     name,
			function: vcsLda2,
			support:  false,
		})
	case "vcsSta3":
		tgt, err = mem.relocateStrongArmFunction(strongArmFunctionSpec{
			name:     name,
			function: vcsSta3,
			support:  false,
		})
	case "SnoopDataBus":
		tgt, err = mem.relocateStrongArmFunction(strongArmFunctionSpec{
			name:     name,
			function: snoopDataBus,
			support:  false,
		})
	case "vcsRead4":
		tgt, err = mem.relocateStrongArmFunction(strongArmFunctionSpec{
			name:     name,
			function: vcsRead4,
			support:  false,
		})
	case "vcsStartOverblank":
		tgt, err = mem.relocateStrongArmFunction(strongArmFunctionSpec{
			name:     name,
			function: vcsStartOverblank,
			support:  false,
		})
	case "vcsEndOverblank":
		tgt, err = mem.relocateStrongArmFunction(strongArmFunctionSpec{
			name:     name,
			function: vcsEndOverblank,
			support:  false,
		})
	case "vcsLdaForBusStuff2":
		tgt, err = mem.relocateStrongArmFunction(strongArmFunctionSpec{
			name:     name,
			function: vcsLdaForBusStuff2,
			support:  false,
		})
	case "vcsLdxForBusStuff2":
		tgt, err = mem.relocateStrongArmFunction(strongArmFunctionSpec{
			name:     name,
			function: vcsLdxForBusStuff2,
			support:  false,
		})
	case "vcsLdyForBusStuff2":
		tgt, err = mem.relocateStrongArmFunction(strongArmFunctionSpec{
			name:     name,
			function: vcsLdyForBusStuff2,
			support:  false,
		})
	case "vcsWrite5":
		tgt, err = mem.relocateStrongArmFunction(strongArmFunctionSpec{
			name:     name,
			function: vcsWrite5,
			support:  false,
		})
	case "vcsWrite6":
		tgt, err = mem.relocateStrongArmFunction(strongArmFunctionSpec{
			name:     name,
			function: vcsWrite6,
			support:  false,
		})
	case "vcsLdx2":
		tgt, err = mem.relocateStrongArmFunction(strongArmFunctionSpec{
			name:     name,
			function: vcsLdx2,
			support:  false,
		})
	case "vcsLdy2":
		tgt, err = mem.relocateStrongArmFunction(strongArmFunctionSpec{
			name:     name,
			function: vcsLdy2,
			support:  false,
		})
	case "vcsSta4":
		tgt, err = mem.relocateStrongArmFunction(strongArmFunctionSpec{
			name:     name,
			function: vcsSta4,
			support:  false,
		})
	case "vcsSax3":
		tgt, err = mem.relocateStrongArmFunction(strongArmFunctionSpec{
			name:     name,
			function: vcsSax3,
			support:  false,
		})
	case "vcsStx3":
		tgt, err = mem.relocateStrongArmFunction(strongArmFunctionSpec{
			name:     name,
			function: vcsStx3,
			support:  false,
		})
	case "vcsStx4":
		tgt, err = mem.relocateStrongArmFunction(strongArmFunctionSpec{
			name:     name,
			function: vcsStx4,
			support:  false,
		})
	case "vcsSty3":
		tgt, err = mem.relocateStrongArmFunction(strongArmFunctionSpec{
			name:     name,
			function: vcsSty3,
			support:  false,
		})
	case "vcsSty4":
		tgt, err = mem.relocateStrongArmFunction(strongArmFunctionSpec{
			name:     name,
			function: vcsSty4,
			support:  false,
		})
	case "vcsJsr6":
		tgt, err = mem.relocateStrongArmFunction(strongArmFunctionSpec{
			name:     name,
			function: vcsJsr6,
			support:  false,
		})
	case "vcsNop2":
		tgt, err = mem.relocateStrongArmFunction(strongArmFunctionSpec{
			name:     name,
			function: vcsNop2,
			support:  false,
		})
	case "vcsNop2n":
		tgt, err = mem.relocateStrongArmFunction(strongArmFunctionSpec{
			name:     name,
			function: vcsNop2n,
			support:  false,
		})
	case "vcsTxs2":
		tgt, err = mem.relocateStrongArmFunction(strongArmFunctionSpec{
			name:     name,
			function: vcsTxs2,
			support:  false,
		})
	case "vcsPha3":
		tgt, err = mem.relocateStrongArmFunction(strongArmFunctionSpec{
			name:     name,
			function: vcsPha3,
			support:  false,
		})
	case "vcsPhp3":
		tgt, err = mem.relocateStrongArmFunction(strongArmFunctionSpec{
			name:     name,
			function: vcsPhp3,
			support:  false,
		})
	case "vcsPla4":
		tgt, err = mem.relocateStrongArmFunction(strongArmFunctionSpec{
			name:     name,
			function: vcsPla4,
			support:  false,
		})
	case "vcsPlp4":
		tgt, err = mem.relocateStrongArmFunction(strongArmFunctionSpec{
			name:     name,
			function: vcsPlp4,
			support:  false,
		})
	case "vcsPla4Ex":
		tgt, err = mem.relocateStrongArmFunction(strongArmFunctionSpec{
			name:     name,
			function: vcsPla4Ex,
			support:  false,
		})
		mem.usesBusStuffing = true
	case "vcsPlp4Ex":
		tgt, err = mem.relocateStrongArmFunction(strongArmFunctionSpec{
			name:     name,
			function: vcsPlp4Ex,
			support:  false,
		})
		mem.usesBusStuffing = true
	case "vcsWaitForAddress":
		tgt, err = mem.relocateStrongArmFunction(strongArmFunctionSpec{
			name:     name,
			function: vcsWaitForAddress,
			support:  false,
		})
	case "vcsJmpToRam3":
		tgt, err = mem.relocateStrongArmFunction(strongArmFunctionSpec{
			name:     name,
			function: vcsJmpToRam3,
			support:  false,
		})
	case "vcsWrite4":
		mem.usesBusStuffing = true
		tgt, err = mem.relocateStrongArmFunction(strongArmFunctionSpec{
			name:     name,
			function: vcsWrite4,
			support:  false,
		})
	case "vcsPokeRomByte":
		tgt, err = mem.relocateStrongArmFunction(strongArmFunctionSpec{
			name:     name,
			function: vcsPokeRomByte,
			support:  false,
		})
	case "vcsSetNextAddress":
		tgt, err = mem.relocateStrongArmFunction(strongArmFunctionSpec{
			name:     name,
			function: vcsSetNextAddress,
			support:  true,
		})

	// C library functions that are often not linked but required
	case "randint":
		tgt, err = mem.relocateStrongArmFunction(strongArmFunctionSpec{
			name:     name,
			function: randint,
			support:  true,
		})
	case "memset", "_aeabi_memset":
		tgt, err = mem.relocateStrongArmFunction(strongArmFunctionSpec{
			name:     name,
			function: memset,
			support:  true,
		})
	case "memcpy", "__aeabi_memcpy":
		tgt, err = mem.relocateStrongArmFunction(strongArmFunctionSpec{
			name:     name,
			function: memcpy,
			support:  true,
		})

	// strongARM tables
	case "ReverseByte":
		tgt = mem.relocateStrongArmTable(reverseByteTable)

	case "ColorLookup":
		switch mem.env.TV.GetFrameInfo().Spec.ID {
		case "PAL":
			tgt = mem.relocateStrongArmTable(palColorTable)
		case "NTSC":
			fallthrough
		default:
			tgt = mem.relocateStrongArmTable(ntscColorTable)
		}

	default:
		return false, 0, nil
	}

	return true, tgt, err
}
