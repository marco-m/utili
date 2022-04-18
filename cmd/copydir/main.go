// Useful when developing CopyDir.
// Based on work I did for github.com/Pix4D/cogito/cmd/copydir/main.go
package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/marco-m/docopt-go"
	"github.com/marco-m/utili"
)

const usage = `copydir -- copy a directory with optional transformations

Usage:
  copydir -h | --help
  copydir [options] <srcdir> <dstdir> [ <keyvals> ... ]

Generic options:
  -h --help     print this help
  -v --verbose  be verbose

Options:
  --dot         rename each dot.something to .something

Arguments
  <keyvals>     is of the form k1=v1 k2=v2 ... and enables Go template processing
`

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "copydir: error:", err)
		os.Exit(1)
	}
}

type config struct {
	Verbose bool
	Dot     bool
	SrcDir  string   `docopt:"<srcdir>"`
	DstDir  string   `docopt:"<dstdir>"`
	KeyVals []string `docopt:"<keyvals>"`
	//
}

func run(args []string) error {
	parser := &docopt.Parser{OptionsFirst: true}
	opts, err := parser.ParseArgs(usage, args, "")
	if err != nil {
		return err
	}

	out := out{verbose: opts["--verbose"].(bool)}
	out.debugf("%v", opts)

	app := &config{}
	if err := opts.Bind(app); err != nil {
		return err
	}

	tmplData, err := makeTemplateData(app.KeyVals)
	if err != nil {
		return err
	}
	rename := utili.IdentityRename
	if app.Dot {
		rename = utili.DotRename
	}

	if err := utili.CopyDir2(app.SrcDir, app.DstDir, rename, tmplData); err != nil {
		return err
	}

	return nil
}

// Take a list of strings of the form "key=value" and convert them to map entries.
func makeTemplateData(keyvals []string) (utili.TemplateData, error) {
	data := utili.TemplateData{}
	for _, kv := range keyvals {
		pos := strings.Index(kv, "=")
		if pos == -1 {
			return data, fmt.Errorf("missing '=' in %s", kv)
		}
		key := kv[:pos]
		value := kv[pos+1:]
		data[key] = value
	}
	return data, nil
}

type out struct {
	verbose bool
}

func (out out) debugf(format string, a ...any) {
	if !out.verbose {
		return
	}
	fmt.Println(fmt.Sprintf(format, a...))
}
