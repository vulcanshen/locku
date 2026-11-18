package main

import (
	"os"
	"os/exec"
	"testing"

	"github.com/vulcanshen/locku/internal/version"
)

// `locku version` prints the version alone, as the tag reads it (`v1.0.1`;
// a local build's `dev`), and `locku help` the usage (tdp E1, harness
// project.4). main runs in a child of the test binary, as a shell runs it.
func TestVersionAndHelp(t *testing.T) {
	if arg := os.Getenv("LOCKU_TEST_MAIN"); arg != "" {
		os.Args = []string{"locku", arg}
		main()
		os.Exit(0)
	}
	for arg, want := range map[string]string{"version": version.Display() + "\n", "help": usage} {
		cmd := exec.Command(os.Args[0], "-test.run=^TestVersionAndHelp$")
		cmd.Env = append(os.Environ(), "LOCKU_TEST_MAIN="+arg)
		out, err := cmd.Output()
		if err != nil || string(out) != want {
			t.Errorf("locku %s: %v\n%q\nwant %q", arg, err, out, want)
		}
	}
}
