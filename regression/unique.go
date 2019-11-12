package regression

import (
	"fmt"
	"gopher2600/cartridgeloader"
	"gopher2600/paths"
	"time"
)

// create a unique filename from a CatridgeLoader instance. used when saving
// scripts into regressionScripts directory
func uniqueFilename(prepend string, cartload cartridgeloader.Loader) string {
	n := time.Now()
	timestamp := fmt.Sprintf("%04d%02d%02d_%02d%02d%02d", n.Year(), n.Month(), n.Day(), n.Hour(), n.Minute(), n.Second())
	newScript := fmt.Sprintf("%s_%s_%s", prepend, cartload.ShortName(), timestamp)
	return paths.ResourcePath(regressionScripts, newScript)
}
