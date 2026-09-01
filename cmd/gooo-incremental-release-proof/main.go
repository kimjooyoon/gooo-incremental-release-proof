package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/kimjooyoon/gooo-incremental-release-proof/internal/proof"
)

func main() {
	if len(os.Args) < 2 || os.Args[1] != "conformance" {
		fatalf("usage: gooo-incremental-release-proof conformance")
	}
	if err := runConformance(os.Args[2:]); err != nil {
		fatalf("%v", err)
	}
}

func runConformance(args []string) error {
	flags := flag.NewFlagSet("conformance", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	meta := flags.String("meta", "", "authoritative .gooo contract")
	corpus := flags.String("corpus", "", "fixed conformance corpus")
	root := flags.String("root", ".", "input repository root")
	out := flags.String("out", "", "caller-owned output directory")
	toolchain := flags.String("toolchain", "", "Go toolchain identity")
	runner := flags.String("runner-digest", "", "CI runner identity digest")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if *meta == "" || *corpus == "" || *out == "" {
		return fmt.Errorf("meta, corpus, and out are required")
	}
	report, err := proof.RunConformance(proof.Options{Root: *root, MetaPath: *meta, CorpusPath: *corpus, OutputPath: *out, Toolchain: *toolchain, RunnerDigest: *runner})
	if err != nil {
		return err
	}
	if err := proof.WriteJSON(*out+"/conformance.json", *root, report); err != nil {
		return err
	}
	if report.Decision != proof.Closed {
		return fmt.Errorf("fixed corpus did not replay exactly: %s", report.Decision)
	}
	fmt.Printf("decision=%s cases=%d closed=%d unknown=%d refuted=%d tests=%d/%d/%d/%d/%d/%d\n", report.Decision, report.Summary.TotalCases, report.Summary.Closed, report.Summary.Unknown, report.Summary.Refuted, report.Summary.TestsTotal, report.Summary.TestsSelected, report.Summary.TestsExecuted, report.Summary.TestsReused, report.Summary.TestsFailed, report.Summary.TestsUnknown)
	return nil
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
