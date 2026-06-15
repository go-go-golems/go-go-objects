// Package doc embeds the Glazed help pages shipped by the durableobjects
// xgoja provider. Both the standalone go-go-objects binary and generated
// xgoja binaries that select the provider's HelpSource read from this single
// embedded source of truth.
package doc

import (
	"embed"
	"io/fs"

	"github.com/go-go-golems/glazed/pkg/help"
)

//go:embed *.md
var docFS embed.FS

// AddDocToHelpSystem loads every Glazed Markdown page embedded in this
// package into the given help system.
func AddDocToHelpSystem(helpSystem *help.HelpSystem) error {
	return helpSystem.LoadSectionsFromFS(docFS, ".")
}

// HelpFS returns the embedded filesystem of Glazed Markdown pages and the
// root directory within it. The returned pair matches the FS/Root shape used
// by providerapi.HelpSource so a provider can ship the docs without rebuilding
// the help system itself.
func HelpFS() (fs.FS, string) {
	sub, err := fs.Sub(docFS, ".")
	if err != nil {
		// fs.Sub on an embed.FS with "." only fails if the embed.FS is nil,
		// which is a programmer error. Fall back to the raw embed.FS rather
		// than returning nothing so callers still get the embedded pages.
		return docFS, "."
	}
	return sub, "."
}
