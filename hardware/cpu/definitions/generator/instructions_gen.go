//go:generate go run instructions_gen.go

package main

import (
	"encoding/csv"
	"fmt"
	"gopher2600/hardware/cpu/definitions"
	"io"
	"os"
	"path"
	"sort"
	"strconv"
	"strings"
)

const definitionsCSVFile = "./instructions.csv"
const generatedGoFile = "../instructions.go"

const leadingBoilerPlate = "// generated code - do not change\n\npackage definitions\n\n// GetInstructionDefinitions returns the opcode table for the MC6502\nfunc GetInstructionDefinitions() ([]*InstructionDefinition, error) {\nreturn []*InstructionDefinition{"
const trailingBoilerPlate = "}, nil\n}"

type opCodes map[uint8]definitions.InstructionDefinition

// parseCSV reads & parses the definitions CSV file and creates & returns a map
// of InstructionDefinitions
func parseCSV() (opCodes, error) {
	// open file
	df, err := os.Open(definitionsCSVFile)
	if err != nil {
		// can't open definitions csv file using full path, so try to open
		// it from the current directory. this allows us to run "go test" on
		// the cpu
		// !!TODO: fix how we deal with paths to external resources
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

	// instructions file can have a variable number of fields per definition.
	// instruction effect field is optional (defaulting to READ)
	csvr.FieldsPerRecord = -1

	// create new definitions table
	deftable := make(opCodes)

	for {
		// loop through file until EOF is reached
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

		// trim trailing comment from record
		rec[len(rec)-1] = strings.Split(rec[len(rec)-1], "#")[0]

		// manually trim trailing space from all fields in the record
		for i := 0; i < len(rec); i++ {
			rec[i] = strings.TrimSpace(rec[i])
		}

		newDef := definitions.InstructionDefinition{}

		// field: parse object code
		objectCode := rec[0]
		if objectCode[:2] == "0x" {
			objectCode = objectCode[2:]
		}
		objectCode = strings.ToUpper(objectCode)

		// store the decimal number in the new instruction definition
		// -- we'll use this for the hash key too
		n, err := strconv.ParseInt(objectCode, 16, 16)
		if err != nil {
			return nil, fmt.Errorf("invalid object code (0x%s)", objectCode)
		}
		newDef.ObjectCode = uint8(n)

		// field: instruction mnemonic
		newDef.Mnemonic = rec[1]

		// field: cycle count
		newDef.Cycles, err = strconv.Atoi(rec[2])
		if err != nil {
			return nil, fmt.Errorf("invalid cycle count for 0x%s (%s)", objectCode, rec[2])
		}

		// field: addressing mode
		//
		// the addressing mode also defines how many bytes an instruction
		// requires
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

		// field: page sensitive
		ps := strings.ToUpper(rec[4])
		switch ps {
		default:
			return nil, fmt.Errorf("invalid page sensitivity switch for 0x%s (%s)", objectCode, rec[4])
		case "TRUE":
			newDef.PageSensitive = true
		case "FALSE":
			newDef.PageSensitive = false
		}

		// field: effect category
		if len(rec) == 5 {
			// effect field is optional. if it hasn't been included then
			// default instruction effect defaults to 'Read'
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
			case "INTERRUPT":
				newDef.Effect = definitions.Interrupt
			}
		}

		// add new definition to deftable, using object code as the hash key
		deftable[newDef.ObjectCode] = newDef
	}

	return deftable, nil
}

func printSummary(deftable opCodes) {
	missing := make([]int, 0, 255)

	// walk deftable and note missing opcodes
	for i := 0; i <= 255; i++ {
		if _, ok := deftable[uint8(i)]; !ok {
			missing = append(missing, i)
		}
	}

	// if no missing opcodes were found then there is nothing more to do
	if len(missing) == 0 {
		return
	}

	fmt.Println("6510 implementation / undefined opcodes")
	fmt.Println("---------------------------------------")

	// sort missing opcodes
	missing = sort.IntSlice(missing)

	// print and columnise missing opcodes
	c := 0
	for i := range missing {
		fmt.Printf("%#02x\t", missing[i])
		c++
		if c > 4 {
			c = 0
			fmt.Printf("\n")
		}
	}
	if c != 0 {
		fmt.Printf("\n")
	}

	// print summary
	fmt.Printf("%d missing, %02.0f%% defined\n", len(missing), float32(100*(256-len(missing))/256))
	fmt.Println("(defined means that the taxonomy of the opcode has been identified, not necessarily implemented)")
}

func main() {
	// parse definitions files
	deftable, err := parseCSV()
	if err != nil {
		fmt.Printf("error during opcode generation (%s)", err)
		os.Exit(10)
	}

	// output the definitions map as an array
	output := ""
	for opcode := 0; opcode < 256; opcode++ {
		def, found := deftable[uint8(opcode)]
		if found {
			output = fmt.Sprintf("%s\n&%#v,", output, def)
		} else {
			output = fmt.Sprintf("%s\nnil,", output)
		}
	}

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

	printSummary(deftable)
}
