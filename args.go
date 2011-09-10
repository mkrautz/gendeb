package main

import (
	"flag"
	"fmt"
	"os"
)

var Args args

type args struct {
	Spec     string
	Version  string
	Out      string
	ShowHelp bool
}

func init() {
	flag.StringVar(&Args.Spec, "spec", "", "The spec file to ues for deb generation")
	flag.StringVar(&Args.Version, "version", "", "Override the version in the spec file")
	flag.StringVar(&Args.Out, "out", "", "Override the output filename for the deb file")
	flag.BoolVar(&Args.ShowHelp, "help", false, "Show this help message")
}

func Usage() {
	fmt.Fprintf(os.Stderr, "usage: gendeb -spec=<specfile> [options]\n")
	flag.PrintDefaults()
}
