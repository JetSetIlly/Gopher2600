package commandline

// Index maps names to entries in a commands table. Allows direct access to
// individual nodes without having to search the list. Useful for stuff like:
//
//	help(index["foo"])
//
type Index map[string]*node

// CreateIndex returns an index of the Commands structure
func CreateIndex(cmds *Commands) *Index {
	idx := make(Index, 0)

	for ci := range *cmds {
		idx[(*cmds)[ci].tag] = (*cmds)[ci]
	}

	return &idx
}
