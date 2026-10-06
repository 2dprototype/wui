// Command wmlcheck validates WML layout files without opening a window.
//
//	wmlcheck login.xml other.xml     check files, exit status 1 on errors
//	wmlcheck -lenient login.xml      unknown properties are only warnings
//	wmlcheck -fmt login.xml          print the canonical form of a file
//	wmlcheck -fmt -w login.xml       rewrite the file in canonical form
//	wmlcheck -schema                 list all elements, properties and events
//
// It works on every operating system because it does not use the Windows API.
package main

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/2dprototype/wui/wml"
)

func main() {
	format := flag.Bool("fmt", false, "print the canonical form of every valid file")
	write := flag.Bool("w", false, "with -fmt: rewrite the files instead of printing them")
	lenient := flag.Bool("lenient", false, "report unknown properties as warnings")
	schema := flag.Bool("schema", false, "print the supported elements, properties and events")
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "usage: wmlcheck [-fmt [-w]] [-lenient] file.xml...")
		fmt.Fprintln(os.Stderr, "       wmlcheck -schema")
		flag.PrintDefaults()
	}
	flag.Parse()

	if *schema {
		printSchema()
		return
	}
	if flag.NArg() == 0 {
		flag.Usage()
		os.Exit(2)
	}
	if *write && !*format {
		fmt.Fprintln(os.Stderr, "wmlcheck: -w only works together with -fmt")
		os.Exit(2)
	}

	var opts []wml.ParseOption
	if *lenient {
		opts = append(opts, wml.Lenient())
	}

	failed := false
	for _, path := range flag.Args() {
		doc, err := wml.ParseFile(path, opts...)
		if err != nil {
			failed = true
			if list, ok := err.(wml.ErrorList); ok {
				fmt.Fprintln(os.Stderr, list.Details())
			} else {
				fmt.Fprintln(os.Stderr, err)
			}
			continue
		}
		for _, w := range doc.Warnings {
			fmt.Fprintf(os.Stderr, "warning: %s\n", w)
		}

		switch {
		case *format && *write:
			out := doc.Marshal()
			old, err := ioutil.ReadFile(path)
			if err == nil && bytes.Equal(old, out) {
				continue
			}
			if err := ioutil.WriteFile(path, out, 0666); err != nil {
				failed = true
				fmt.Fprintln(os.Stderr, err)
			}
		case *format:
			os.Stdout.Write(doc.Marshal())
		default:
			fmt.Printf("%s: ok\n", path)
		}
	}
	if failed {
		os.Exit(1)
	}
}

func printSchema() {
	for _, name := range wml.TypeNames() {
		t := wml.Lookup(name)
		kind := ""
		if t.Container {
			kind = "  (can contain controls)"
		}
		fmt.Printf("%s%s\n", t.Name, kind)
		fmt.Println("  Name  identifier (optional)")
		for _, p := range t.Props {
			fmt.Printf("  %s\n", p.Describe())
		}
		for _, e := range t.Events {
			fmt.Printf("  %s  handler name\n", e)
		}
		fmt.Println()
	}
	fmt.Println("Font  (child of any element)")
	fmt.Println("  Name  string")
	fmt.Println("  Height  int")
	fmt.Println("  Bold, Italic, Underlined, StrikedOut  bool")
}
