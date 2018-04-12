package cpu

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

const definitionsCSVFile = "./definitions.csv"

// AddressingMode describes the method by which an instruction receives data
// on which to operate
type AddressingMode int

// enumeration of supported addressing modes
const (
	Implied AddressingMode = iota
	Immediate
	Relative // relative addressing is used for branch instructions

	Absolute // sometimes called absolute addressing
	ZeroPage
	Indirect // indirect addressing (with no indexing) is only for JMP instructions

	PreIndexedIndirect  // uses X register
	PostIndexedIndirect // uses Y register
	AbsoluteIndexedX
	AbsoluteIndexedY
	IndexedZeroPageX
	IndexedZeroPageY // only used for LDX
)

// EffectCategory - categorises an instruction by the effect it has
type EffectCategory int

// enumeration of instruction effect categories
const (
	Read EffectCategory = iota
	Write
	RMW
	Flow
	Subroutine
)

// InstructionDefinition type is the property list for each instruction
type InstructionDefinition struct {
	ObjectCode     uint8
	Mnemonic       string
	Bytes          int
	Cycles         int
	AddressingMode AddressingMode
	PageSensitive  bool
	Effect         EffectCategory
}

type definitionsTable map[uint8]InstructionDefinition

func getInstructionDefinitions() (definitionsTable, error) {
	df, err := os.Open(definitionsCSVFile)
	if err != nil {
		return nil, fmt.Errorf("error opening instruction definitions (%s)", err)
	}
	defer func() {
		_ = df.Close()
	}()

	// treat the file as a CSV file
	csvr := csv.NewReader(df)
	csvr.Comment = rune('#')
	csvr.TrimLeadingSpace = true
	csvr.ReuseRecord = true

	// csv file can have a variable number of fields per record
	csvr.FieldsPerRecord = -1

	// create new definitions table
	definitions := make(definitionsTable)

	for {
		rec, err := csvr.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		// check for valid record length
		if !(len(rec) == 5 || len(rec) == 6) {
			return nil, fmt.Errorf("wrong number of fields in instruction definition (%s)", rec)
		}

		newDef := InstructionDefinition{}

		// parse object code -- we'll use this for the hash key too
		objectCode := rec[0]
		if objectCode[:2] == "0x" {
			objectCode = objectCode[2:]
		}
		objectCode = strings.ToUpper(objectCode)

		// store the decimal number in the hash table
		n, err := strconv.ParseInt(objectCode, 16, 16)
		if err != nil {
			return nil, fmt.Errorf("invalid object code (0x%s)", objectCode)
		}
		newDef.ObjectCode = uint8(n)

		// instruction mnemonic
		newDef.Mnemonic = rec[1]

		// cycle count
		newDef.Cycles, err = strconv.Atoi(rec[2])
		if err != nil {
			return nil, fmt.Errorf("invalid cycle count for 0x%s (%s)", objectCode, rec[2])
		}

		// addressing Mode - also taking the opportunity to record the number of bytes used
		// by the instruction - inferred from the addressing mode
		am := strings.ToUpper(rec[3])
		switch am {
		default:
			return nil, fmt.Errorf("invalid addressing mode for 0x%s (%s)", objectCode, rec[3])
		case "IMPLIED":
			newDef.AddressingMode = Implied
			newDef.Bytes = 1
		case "IMMEDIATE":
			newDef.AddressingMode = Immediate
			newDef.Bytes = 2
		case "RELATIVE":
			newDef.AddressingMode = Relative
			newDef.Bytes = 2
		case "ABSOLUTE":
			newDef.AddressingMode = Absolute
			newDef.Bytes = 3
		case "ZERO_PAGE":
			newDef.AddressingMode = ZeroPage
			newDef.Bytes = 2
		case "INDIRECT":
			newDef.AddressingMode = Indirect
			newDef.Bytes = 3
		case "PRE_INDEX_INDIRECT":
			newDef.AddressingMode = PreIndexedIndirect
			newDef.Bytes = 2
		case "POST_INDEX_INDIRECT":
			newDef.AddressingMode = PostIndexedIndirect
			newDef.Bytes = 2
		case "ABSOLUTE_INDEXED_X":
			newDef.AddressingMode = AbsoluteIndexedX
			newDef.Bytes = 3
		case "ABSOLUTE_INDEXED_Y":
			newDef.AddressingMode = AbsoluteIndexedY
			newDef.Bytes = 3
		case "INDEXED_ZERO_PAGE_X":
			newDef.AddressingMode = IndexedZeroPageX
			newDef.Bytes = 2
		case "INDEXED_ZERO_PAGE_Y":
			newDef.AddressingMode = IndexedZeroPageY
			newDef.Bytes = 2
		}

		// page sensitive
		ps := strings.ToUpper(rec[4])
		switch ps {
		default:
			return nil, fmt.Errorf("invalid page sensitivity switch for 0x%s (%s)", objectCode, rec[4])
		case "TRUE":
			newDef.PageSensitive = true
		case "FALSE":
			newDef.PageSensitive = false
		}

		// effect category
		if len(rec) == 5 {
			// default category
			newDef.Effect = Read
		} else {
			switch rec[5] {
			default:
				return nil, fmt.Errorf("unknown category for 0x%s (%s)", objectCode, rec[5])
			case "READ":
				newDef.Effect = Read
			case "WRITE":
				newDef.Effect = Write
			case "RMW":
				newDef.Effect = RMW
			case "FLOW":
				newDef.Effect = Flow
			case "SUB-ROUTINE":
				newDef.Effect = Subroutine
			}
		}

		// insert new definition into the table
		definitions[newDef.ObjectCode] = newDef
	}

	return definitions, nil
}
