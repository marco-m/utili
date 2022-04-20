package utili

import (
	"bytes"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"testing"
)

// Passed to template.Execute()
type TemplateData map[string]string

type RenameFn func(string) string

// DotRename returns a copy of `name` with the prefix "dot." replaced by ".", otherwise
// returns `name` as-is.
// Example: "dot.git" => ".git".
// Useful to store in the testdata directory git repositories (as "dot.git") while
// avoiding git complaining about the fact that testdata/.git is a nested git repo.
// Meant to be passed to function CopyDir.
//
// Based on work I did for github.com/Pix4D/cogito/help/testhelper.go
func DotRename(name string) string {
	return strings.Replace(name, "dot.", ".", 1)
}

// IdentityRename returns `name` unchanged.
// Meant to be passed to function CopyDir.
func IdentityRename(name string) string {
	return name
}

// CopyDir recursively copies the `src` directory below the `dst` directory, with
// optional transformations.
// It performs the following transformations:
// - Renames any directory by applying `rename` to it.
// - If `tmplData` is not empty, will treat each file ending with ".template" as a Go
//   template and fill it accordingly.
// - If a file name contains basic Go template formatting (eg: `foo-{{.bar}}.template`),
//   then the file will be renamed accordingly.
//
// It will fail if the dst directory doesn't exist.
//
// For example, if src directory is `foo`:
//
// foo
// └── dot.git
//     └── config
//
// and dst directory is `bar`, src will be copied as:
//
// bar
// └── foo
//     └── .git        <= dot renamed
//         └── config
//
// See also CopyDir2 for usage outside a testing environment.
//
// Based on work I did for github.com/Pix4D/cogito/help/testhelper.go
func CopyDir(
	t *testing.T,
	src string,
	dst string,
	rename RenameFn,
	tmplData TemplateData,
) {
	t.Helper()

	if err := CopyDir2(src, dst, rename, tmplData); err != nil {
		t.Fatal("CopyDir:", err)
	}
}

func CopyDir2(src string, dst string, rename RenameFn, tmplData TemplateData) error {
	for _, dir := range []string{src, dst} {
		fi, err := os.Stat(dir)
		if err != nil {
			return err
		}
		if !fi.IsDir() {
			return fmt.Errorf("%v is not a directory", dst)
		}
	}

	renamedDir := rename(filepath.Base(src))
	tgtDir := filepath.Join(dst, renamedDir)
	if err := os.MkdirAll(tgtDir, 0770); err != nil {
		return fmt.Errorf("making dst dir: %s", err)
	}

	srcEntries, err := ioutil.ReadDir(src)
	if err != nil {
		return err
	}
	for _, e := range srcEntries {
		src := filepath.Join(src, e.Name())
		if e.IsDir() {
			if err := CopyDir2(src, tgtDir, rename, tmplData); err != nil {
				return err
			}
		} else {
			name := e.Name()
			if len(tmplData) != 0 {
				// FIXME longstanding bug: we apply template processing always, also if the file
				// doesn't have the .template suffix!
				name = strings.TrimSuffix(name, ".template")
				// Subject the file name itself to template expansion
				tmpl, err := template.New("file-name").Parse(name)
				if err != nil {
					return fmt.Errorf("parsing file name as template %v: %w", src, err)
				}
				tmpl.Option("missingkey=error")
				buf := &bytes.Buffer{}
				if err := tmpl.Execute(buf, tmplData); err != nil {
					return fmt.Errorf("executing template file name %v with data %v: %w",
						src, tmplData, err)
				}
				name = buf.String()
			}
			if err := copyFile(src, filepath.Join(tgtDir, name), tmplData); err != nil {
				return err
			}
		}

	}
	return nil
}

func copyFile(srcPath string, dstPath string, tmplData TemplateData) error {
	srcFile, err := os.Open(srcPath)
	if err != nil {
		return fmt.Errorf("opening src file: %w", err)
	}
	defer srcFile.Close()

	// We want an error if the file already exists
	dstFile, err := os.OpenFile(dstPath, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0660)
	if err != nil {
		return fmt.Errorf("creating dst file: %w", err)
	}
	defer dstFile.Close()

	if len(tmplData) == 0 {
		_, err = io.Copy(dstFile, srcFile)
		return err
	}
	buf, err := ioutil.ReadAll(srcFile)
	if err != nil {
		return err
	}
	tmpl, err := template.New(path.Base(srcPath)).Parse(string(buf))
	if err != nil {
		return fmt.Errorf("parsing template %v: %w", srcPath, err)
	}
	tmpl.Option("missingkey=error")
	if err := tmpl.Execute(dstFile, tmplData); err != nil {
		return fmt.Errorf("executing template %v with data %v: %w", srcPath, tmplData, err)
	}
	return nil
}

// Tree uses t.Log to print the output of the tree -a utility
func Tree(t *testing.T, dir string) {
	t.Helper()
	out, err := exec.Command("tree", "-a", dir).Output()
	if err != nil {
		t.Fatal("Tree:", err)
	}
	t.Logf("\n%s\n", string(out))
}

// Chdir calls os.Chdir(dir) for the test to use. The directory is restored to the
// previous one by t.Cleanup when the test completes.
// If any operation fails, ChDir terminates the test by calling t.Fatal.
func Chdir(t *testing.T, dir string) {
	t.Helper()

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal("chdir: getting cwd:", err)
	}

	t.Cleanup(func() {
		if err := os.Chdir(cwd); err != nil {
			t.Fatal("chdir: cleanup: doing chdir:", err)
		}
	})

	if err := os.Chdir(dir); err != nil {
		t.Fatal("chdir: doing chdir:", err)
	}
}
