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

// Package commandline facilitates parsing of command line input. Given a
// command template, it can be used to tokenisze and validate user input. It
// also functions as a tab-completion engine, implementing the
// terminal.TabCompletion interface.
//
// The Commands type is the base product of the package. To create an instance
// of Commands, use ParseCommandTemplate() with a suitable template. See the
// function documentation for syntax. An example template would be:
//
//	template := []string {
//		"LIST",
//		"PRINT [%s]",
//		"SORT (RISING|FALLING)",
//	}
//
// Once parsed, the resulting Commands instance can be used to validate input.
//
//	cmds, _ := ParseCommandTemplate(template)
//	toks := TokeniseInput("list")
//	err := cmds.ValidateTokens(toks)
//	if err != nil {
//		panic("validation failed")
//	}
//
// Note that all validation is case-insensitive. Once validated the tokens can
// be processed and acted upon. The commandline package proveds some useful
// functions to work on tokenised input. We've already seen TokeniseInput().
// This function creates an instance of type Tokens. The Get() function can be
// used to retrieve the next token in line.
//
// The beauty of validating tokens against the command template is that we can
// simplify and restrict our handling of Get() returned values to only those
// that we know have passed the validation. For example, using the above
// template, we can implement a switch very consisely:
//
//	option, _ := toks.Get()
//	switch strings.ToUpper(option) {
//		case "LIST:
//			list()
//		case "PRINT:
//			fmt.Println(toks.Get())
//		case "SORT:
//			rising = true
//			if toks.Get() == "FALLING" {
//				rising = false
//			}
//			sort(data, rising)
//	}
//
// # Tab Completion
//
// The TabCompletion type is used to transform input such that it more closely
// resemebles a valid command according to the supplied template. The
// NewTabCompletion() function expects an instance of Commands.
//
//	tbc := NewTabCompletion(cmds)
//
// The Complete() function can then be used to transform user input:
//
//	inp := "LIS"
//	inp = tbc.Complete(inp)
//
// In this instance the value of inp will be "LIST " (note the trailing space).
// Given a number of options to use for the completion, the first option will
// be returned first followed by the second, third, etc. on subsequent calls to
// Complete(). A tab completion session can be terminated with a call to
// Reset().
//
// # Extensions
//
// Extensions are a way of giving a command additional arguments that are not in
// the original template. This is good for handling specialist arguments that
// are rarely used but none-the-less would benefit from tab-completion.
//
// To give a command an extension the %x directive is used. The placeholder must
// have a label for it to be effective. In the example below the label is
// "listing". Note that labels are case insensitive.
//
//	template := []string {
//		"LIST (%<listing>x)",
//		"PRINT [%s]",
//		"SORT (RISING|FALLING)",
//	}
//
// Once the template has been parsed, an extension handler must be added with
// the AddExtension() function. In addition to the label argument, which must be
// the same as the label given in the template, the AddExtension() function also
// requires an implementation of the interface.
//
// The Commands() function of  the Extension interface will return another
// instance of the Commands type. This instance will be used when the %x
// directive is encounted during validation or tab completion.
package commandline
