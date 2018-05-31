package ui

// TabCompleter defines the types that can be used for tab completion
type TabCompleter interface {
	GuessWord(string) string
}
