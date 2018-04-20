//go:generate go run definitions_gen.go

package main

import (
	"encoding/csv"
	"fmt"
	"headlessVCS/hardware/cpu/definitions"
	"io"
	"os"
	"path"
	"strconv"
	"strings"
)

const definitionsCSVFile = "./definitions.csv"
const generatedGoFile = "../instructions.go"

const leadingBoilerPlate = "// generated code - do not change\n\npackage definitions\n\n// GetInstructionDefinitions returns the opcode table for the MC6502\nfunc GetInstructionDefinitions() (map[uint8]InstructionDefinition, error) {\nreturn map[uint8]InstructionDefinition"
const trailingBoilerPlate = ", nil\n}"

func generate() (map[uint8]definitions.InstructionDefinition, error) {
	df, err := os.Open(definitionsCSVFile)
	if err != nil {
		// can't open definitions csv file using full path, so try to open
		// it from the current directory. this allows us to run "go test" on
		// the cpu
		// TODO: fix how we deal with paths to external resources
		_, fn := path.Split(definitionsCSVFile)
		df, err = os.Open(fn)
		if err != nil {
			return nil, fmt.Errorf("error opening instruction definitions (%s)", err)
		}
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
	deftable := make(map[uint8]definitions.InstructionDefinition)

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

		newDef := definitions.InstructionDefinition{}

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
			newDef.AddressingMode = definitions.Implied
			newDef.Bytes = 1
		case "IMMEDIATE":
			newDef.AddressingMode = definitions.Immediate
			newDef.Bytes = 2
		case "RELATIVE":
			newDef.AddressingMode = definitions.Relative
			newDef.Bytes = 2
		case "ABSOLUTE":
			newDef.AddressingMode = definitions.Absolute
			newDef.Bytes = 3
		case "ZERO_PAGE":
			newDef.AddressingMode = definitions.ZeroPage
			newDef.Bytes = 2
		case "INDIRECT":
			newDef.AddressingMode = definitions.Indirect
			newDef.Bytes = 3
		case "PRE_INDEX_INDIRECT":
			newDef.AddressingMode = definitions.PreIndexedIndirect
			newDef.Bytes = 2
		case "POST_INDEX_INDIRECT":
			newDef.AddressingMode = definitions.PostIndexedIndirect
			newDef.Bytes = 2
		case "ABSOLUTE_INDEXED_X":
			newDef.AddressingMode = definitions.AbsoluteIndexedX
			newDef.Bytes = 3
		case "ABSOLUTE_INDEXED_Y":
			newDef.AddressingMode = definitions.AbsoluteIndexedY
			newDef.Bytes = 3
		case "INDEXED_ZERO_PAGE_X":
			newDef.AddressingMode = definitions.IndexedZeroPageX
			newDef.Bytes = 2
		case "INDEXED_ZERO_PAGE_Y":
			newDef.AddressingMode = definitions.IndexedZeroPageY
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
			newDef.Effect = definitions.Read
		} else {
			switch rec[5] {
			default:
				return nil, fmt.Errorf("unknown category for 0x%s (%s)", objectCode, rec[5])
			case "READ":
				newDef.Effect = definitions.Read
			case "WRITE":
				newDef.Effect = definitions.Write
			case "RMW":
				newDef.Effect = definitions.RMW
			case "FLOW":
				newDef.Effect = definitions.Flow
			case "SUB-ROUTINE":
				newDef.Effect = definitions.Subroutine
			}
		}

		// insert new definition into the table
		deftable[newDef.ObjectCode] = newDef
	}

	return deftable, nil
}

func main() {
	deftable, err := generate()
	if err != nil {
		fmt.Printf("error during opcode generation (%s)", err)
		os.Exit(10)
	}

	// use Go syntax representation of deftable as basis of output
	output := fmt.Sprintf("%#v", deftable)

	// trim leading information, up to first brace
	i := strings.Index(output, "{")
	if i == -1 {
		fmt.Printf("error during opcode generation (deftable malformed)")
		os.Exit(10)
	}
	output = output[i:]

	// we'll be putting the contents of deftable into the definition package so
	// we need to remove the expicit references to that package
	output = strings.Replace(output, "definitions.", "", -1)

	// add boiler-plate to output
	output = fmt.Sprintf("%s%s%s", leadingBoilerPlate, output, trailingBoilerPlate)

	// create output file (over-writing) if it already exists
	f, err := os.Create(generatedGoFile)
	if err != nil {
		fmt.Printf("error during opcode generation (%s)", err)
		os.Exit(10)
	}

	_, err = f.WriteString(output)
	if err != nil {
		_ = f.Close()
		fmt.Printf("error during opcode generation (%s)", err)
		os.Exit(10)
	}
}
