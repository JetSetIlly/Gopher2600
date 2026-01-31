package main

import (
	_ "embed"
	"encoding/json"
	"testing"
)

//go:embed "definitions.json"
var definitions_json []byte

// simple test to make sure we can load the definitions
func TestLoad(t *testing.T) {
	var definitions []Instruction
	json.Unmarshal(definitions_json, &definitions)
	_ = definitions
}
