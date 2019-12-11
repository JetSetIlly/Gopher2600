// Package symbols helps keep track of address symbols. The primary structure
// for this is the Table type. There are two recommended ways of instantiating
// this type. NewTable() will create a table instance with the default or
// canonical Atari 2600 symbol names. For example, AUDC0 refers to the $15
// write address.
//
// The second and more flexible way of instantiating the symbols Table is with
// the ReadSymbolsFile() function. This function will try to read a symbols
// file for the named cartridge and parse its contents. It will fail silently
// if it cannot.
//
// ReadSymbolFile() will always give addresses the default or canonised symbol.
// In this way it is a superset of the NewTable() function.
package symbols
