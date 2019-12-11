// Package disassembly coordinates the disassembly of cartridge memory. For
// simple presentations of a cartridge the FromCartridge() function can be
// used. Many debuggers will probably find it more useful to disassemble from
// the memory of an already instantiated VCS.
//
//	disasm, _ := disassembly.FromMemory(cartMem, symbols.NewTable())
//
// The FromMemory() function requires a valid instance of a symbols.Table. In
// the example above, we've simply sent the empty table; which is fine but
// limits the potential of the disassembly package. For best results, the
// symbols.ReadSymbolsFile() function should be used (see symbols package for
// details). Note that the FromCartridge() function handles symbols files for
// you.
//
// The disassembly package performs two types of disassembly: what we call
// linear and flow disasseblies. Both are useful and eack eke out different
// information from cartridge memory. In a nutshell:
//
// Linear disassembly decodes every possible address in the cartridge. if the
// "execution" of the address succeeds it is stored in the linear table. Flow
// disassembly on the other hand decodes only those cartridge addresses that
// flow from the start adddress as the executable program unfolds.
//
// In flow disassembly it is hoped that every branch and subroutine is
// considered. However, it is possible for real execution of the ROM to reach
// places not reacable by the flow. For example:
//
// - Addresses stuffed into the stack and RTS being called, without an explicit
// JSR.
//
// - Branching or jumping to non-cartridge memory. (ie. RAM) and executing code
// there.
//
// The Analysis() function summarises any oddities like these that have been
// detected.
//
// Compared to flow disassembly, linear disassembly looks at every memory
// location. The downside of this is that a lot of what is found will be
// nonsense.  We can use the IsInstruction() function of the Entry type to help
// us decide what is what but none-the-less linear disassembly is no good for
// presenting the entire program. Where linear disassembly *is* useful is a
// quick reference for an address that you know contains a valid instruction.
//
// The flow/linear difference is invisible to the user of the disassembly
// package. Instead, the functions Get(), Dump() and Grep() are used. These
// functions use the most appropriate disassembly for the use case.
//
//	Dump() --> flow
//	Get()  --> linear
//	Grep() --> flow
package disassembly
