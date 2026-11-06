// Command tenant-document-inventory privately inspects a frozen local starter backup.
// It never boots the server, opens a database, loads credentials, or creates a root.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/elcomware/edupilot-os/modules/documents/adapters/outbound/blob"
	"github.com/elcomware/edupilot-os/platform/tenantupgrade"
)

const usage = "Usage: tenant-document-inventory -manifest=PRIVATE_JSON -root=EXISTING_PRIVATE_ROOT [-expected-sha256=APPROVED_DIGEST] [-references=PRIVATE_JSON]"

func run(args []string, output, diagnostics io.Writer) int {
	flags := flag.NewFlagSet("tenant-document-inventory", flag.ContinueOnError)
	// flag errors can echo arbitrary input (including paths/secrets); replace them.
	flags.SetOutput(io.Discard)
	manifestPath := flags.String("manifest", "", "private approved version 1 mapping manifest")
	referencesPath := flags.String("references", "", "private preserved version 1 object reference manifest")
	root := flags.String("root", "", "existing private frozen local document root")
	expected := flags.String("expected-sha256", "", "optional approved aggregate SHA-256")
	if err := flags.Parse(args); err != nil {
		if err == flag.ErrHelp {
			fmt.Fprintln(diagnostics, usage)
			return 0
		}
		fmt.Fprintln(diagnostics, "Invalid inventory command. Use -help for usage.")
		return 2
	}
	explicitExpected := false
	explicitReferences := false
	flags.Visit(func(option *flag.Flag) {
		if option.Name == "expected-sha256" {
			explicitExpected = true
		}
		if option.Name == "references" {
			explicitReferences = true
		}
	})
	if explicitExpected && *expected == "" {
		fmt.Fprintln(diagnostics, "Expected inventory digest must not be empty.")
		return 2
	}
	if *manifestPath == "" || *root == "" || flags.NArg() != 0 {
		fmt.Fprintln(diagnostics, usage)
		return 2
	}
	if explicitReferences && *referencesPath == "" {
		fmt.Fprintln(diagnostics, "Object reference manifest must not be empty.")
		return 2
	}
	if *referencesPath != "" && !explicitExpected {
		fmt.Fprintln(diagnostics, "Reconciliation requires an approved inventory digest.")
		return 2
	}
	info, err := os.Lstat(*manifestPath)
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm()&0077 != 0 {
		fmt.Fprintln(diagnostics, "Cannot read a private regular manifest.")
		return 1
	}
	file, err := os.Open(*manifestPath)
	if err != nil {
		fmt.Fprintln(diagnostics, "Cannot read a private regular manifest.")
		return 1
	}
	defer file.Close()
	opened, err := file.Stat()
	if err != nil || !os.SameFile(info, opened) {
		fmt.Fprintln(diagnostics, "Cannot read a private regular manifest.")
		return 1
	}
	manifest, err := tenantupgrade.ReadManifest(file)
	if err != nil {
		fmt.Fprintln(diagnostics, "Invalid approved mapping manifest.")
		return 1
	}
	var result any
	if *referencesPath == "" {
		result, err = blob.InventoryLocal(context.Background(), *root, manifest, *expected)
		if err != nil {
			fmt.Fprintln(diagnostics, "Local document inventory failed; check the frozen private backup, approved mapping and expected digest.")
			return 1
		}
	} else {
		referenceInfo, statErr := os.Lstat(*referencesPath)
		if statErr != nil || !referenceInfo.Mode().IsRegular() || referenceInfo.Mode().Perm()&0077 != 0 {
			fmt.Fprintln(diagnostics, "Cannot read a private regular object reference manifest.")
			return 1
		}
		referenceFile, openErr := os.Open(*referencesPath)
		if openErr != nil {
			fmt.Fprintln(diagnostics, "Cannot read a private regular object reference manifest.")
			return 1
		}
		defer referenceFile.Close()
		referenceOpened, statErr := referenceFile.Stat()
		if statErr != nil || !os.SameFile(referenceInfo, referenceOpened) {
			fmt.Fprintln(diagnostics, "Cannot read a private regular object reference manifest.")
			return 1
		}
		references, readErr := tenantupgrade.ReadObjectReferences(referenceFile)
		if readErr != nil {
			fmt.Fprintln(diagnostics, "Invalid preserved object reference manifest.")
			return 1
		}
		result, err = blob.ReconcileLocal(context.Background(), *root, manifest, references, *expected)
		if err != nil {
			fmt.Fprintln(diagnostics, "Local document reconciliation failed; check the frozen private backup, preserved references, approved mapping and inventory digest.")
			return 1
		}
	}
	// Marshal the complete aggregate only after every source entry has passed.
	encoded, err := json.Marshal(result)
	if err != nil {
		return 1
	}
	encoded = append(encoded, '\n')
	if written, err := output.Write(encoded); err != nil || written != len(encoded) {
		fmt.Fprintln(diagnostics, "Cannot write private inventory output.")
		return 1
	}
	return 0
}
func main() { os.Exit(run(os.Args[1:], os.Stdout, os.Stderr)) }
