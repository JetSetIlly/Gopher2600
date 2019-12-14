//go:generate go run instructions_gen.go

package main

import (
	"encoding/csv"
	"fmt"
	"go/format"
	"gopher2600/hardware/cpu/instructions"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"
)

const definitionsCSVFile = "./instructions.csv"
const generatedGoFile = "../table.go"

const leadingBoilerPlate = "// generated code - do not change\n\n" +
	"package instructions\n\n" +
	"// GetDefinitions returns the table of instruction definitions for the 6507\n" +
	"func GetDefinitions() ([]*Definition, error) {\n" +
	"return []*Definition{"

const trailingBoilerPlate = "}, nil\n}"

func parseCSV() (string, error) {
	// open file
	df, err := os.Open(definitionsCSVFile)
	if err != nil {
		return "", fmt.Errorf("error opening instruction definitions (%s)", err)
	}
	defer df.Close()

	// treat the file as a CSV file
	csvr := csv.NewReader(df)
	csvr.Comment = rune('#')
	csvr.TrimLeadingSpace = true
	csvr.ReuseRecord = true

	// instruction file can have a variable number of fields per definition.
	// instruction effect field is optional (defaulting to READ)
	csvr.FieldsPerRecord = -1

	// create new definitions table
	deftable := make(map[uint8]instructions.Definition)

	line := 0
	for {
		// loop through file until EOF is reached
		line++
		rec, err := csvr.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}

		// check for valid record length
		if !(len(rec) == 5 || len(rec) == 6) {
			return "", fmt.Errorf("wrong number of fields in instruction definition (%s) [line %d]", rec, line)
		}

		// trim trailing comment from record
		rec[len(rec)-1] = strings.Split(rec[len(rec)-1], "#")[0]

		// manually trim trailing space from all fields in the record
		for i := 0; i < len(rec); i++ {
			rec[i] = strings.TrimSpace(rec[i])
		}

		newDef := instructions.Definition{}

		// field: parse opcode
		opcode := rec[0]
		if opcode[:2] == "0x" {
			opcode = opcode[2:]
		}
		opcode = strings.ToUpper(opcode)

		// store the decimal number in the new instruction definition
		// -- we'll use this for the hash key too
		n, err := strconv.ParseInt(opcode, 16, 16)
		if err != nil {
			return "", fmt.Errorf("invalid opcode (%#02x) [line %d]", opcode, line)
		}
		newDef.OpCode = uint8(n)

		// field: opcode mnemonic
		newDef.Mnemonic = rec[1]

		// field: cycle count
		newDef.Cycles, err = strconv.Atoi(rec[2])
		if err != nil {
			return "", fmt.Errorf("invalid cycle count for %#02x (%s) [line %d]", newDef.OpCode, rec[2], line)
		}

		// field: addressing mode
		//
		// the addressing mode also defines how many bytes an opcode
		// requires
		am := strings.ToUpper(rec[3])
		switch am {
		default:
			return "", fmt.Errorf("invalid addressing mode for %#02x (%s) [line %d]", newDef.OpCode, rec[3], line)
		case "IMPLIED":
			newDef.AddressingMode = instructions.Implied
			newDef.Bytes = 1
		case "IMMEDIATE":
			newDef.AddressingMode = instructions.Immediate
			newDef.Bytes = 2
		case "RELATIVE":
			newDef.AddressingMode = instructions.Relative
			newDef.Bytes = 2
		case "ABSOLUTE":
			newDef.AddressingMode = instructions.Absolute
			newDef.Bytes = 3
		case "ZERO_PAGE":
			newDef.AddressingMode = instructions.ZeroPage
			newDef.Bytes = 2
		case "INDIRECT":
			newDef.AddressingMode = instructions.Indirect
			newDef.Bytes = 3
		case "PRE_INDEX_INDIRECT":
			newDef.AddressingMode = instructions.PreIndexedIndirect
			newDef.Bytes = 2
		case "POST_INDEX_INDIRECT":
			newDef.AddressingMode = instructions.PostIndexedIndirect
			newDef.Bytes = 2
		case "ABSOLUTE_INDEXED_X":
			newDef.AddressingMode = instructions.AbsoluteIndexedX
			newDef.Bytes = 3
		case "ABSOLUTE_INDEXED_Y":
			newDef.AddressingMode = instructions.AbsoluteIndexedY
			newDef.Bytes = 3
		case "INDEXED_ZERO_PAGE_X":
			newDef.AddressingMode = instructions.IndexedZeroPageX
			newDef.Bytes = 2
		case "INDEXED_ZERO_PAGE_Y":
			newDef.AddressingMode = instructions.IndexedZeroPageY
			newDef.Bytes = 2
		}

		// field: page sensitive
		ps := strings.ToUpper(rec[4])
		switch ps {
		default:
			return "", fmt.Errorf("invalid page sensitivity switch for %#02x (%s) [line %d]", newDef.OpCode, rec[4], line)
		case "TRUE":
			newDef.PageSensitive = true
		case "FALSE":
			newDef.PageSensitive = false
		}

		// field: effect category
		if len(rec) == 5 {
			// effect field is optional. if it hasn't been included then
			// default instruction effect defaults to 'Read'
			newDef.Effect = instructions.Read
		} else {
			switch rec[5] {
			default:
				return "", fmt.Errorf("unknown category for %#02x (%s) [line %d]", newDef.OpCode, rec[5], line)
			case "READ":
				newDef.Effect = instructions.Read
			case "WRITE":
				newDef.Effect = instructions.Write
			case "RMW":
				newDef.Effect = instructions.RMW
			case "FLOW":
				newDef.Effect = instructions.Flow
			case "SUB-ROUTINE":
				newDef.Effect = instructions.Subroutine
			case "INTERRUPT":
				newDef.Effect = instructions.Interrupt
			}
		}

		// add new definition to deftable, using opcode as the hash key
		deftable[newDef.OpCode] = newDef
	}

	printSummary(deftable)

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

	return output, nil
}

func printSummary(deftable map[uint8]instructions.Definition) {
	missing := make([]int, 0, 255)

	// walk deftable and note missing instructions
	for i := 0; i <= 255; i++ {
		if _, ok := deftable[uint8(i)]; !ok {
			missing = append(missing, i)
		}
	}

	// if no missing instructions were found then there is nothing more to do
	if len(missing) == 0 {
		return
	}

	fmt.Println("6507 implementation / unused opcodes")
	fmt.Println("------------------------------------")

	// sort missing instructions
	missing = sort.IntSlice(missing)

	// print and columnise missing instructions
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
	fmt.Println("(defined means that the taxonomy of the instruction\nhas been identified, not necessarily implemented)")
}

func main() {
	// parse definitions files
	output, err := parseCSV()
	if err != nil {
		fmt.Printf("error during instruction table generation: %s\n", err)
		os.Exit(10)
	}

	// we'll be putting the contents of deftable into the definition package so
	// we need to remove the expicit references to that package
	output = strings.Replace(output, "instructions.", "", -1)

	// add boiler-plate to output
	output = fmt.Sprintf("%s%s%s", leadingBoilerPlate, output, trailingBoilerPlate)

	// format code using standard Go formatted
	formattedOutput, err := format.Source([]byte(output))
	if err != nil {
		fmt.Printf("error during instruction table generation: %s\n", err)
		os.Exit(10)
	}
	output = string(formattedOutput)

	// create output file (over-writing) if it already exists
	f, err := os.Create(generatedGoFile)
	if err != nil {
		fmt.Printf("error during instruction table generation: %s\n", err)
		os.Exit(10)
	}
	defer f.Close()

	_, err = f.WriteString(output)
	if err != nil {
		fmt.Printf("error during instruction table generation: %s\n", err)
		os.Exit(10)
	}
}
