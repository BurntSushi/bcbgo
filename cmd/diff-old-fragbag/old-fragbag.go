package main

import (
	"fmt"
	"os/exec"
	"strings"
)

func runOldFragbag(libFile, pdbFile string, size, fraglen int) (string, error) {
	cmd := []string{
		flagFragbag,
		"-l", libFile,
		fmt.Sprintf("%d", size),
		"-z", fmt.Sprintf("%d", fraglen),
		"-p", pdbFile,
		"-c"}
	out, err := exec.Command(cmd[0], cmd[1:]...).CombinedOutput()
	if err != nil {
		return strings.TrimSpace(string(out)),
			fmt.Errorf("There was an error executing: %s\n%s",
				strings.Join(cmd, " "), err)
	}
	return strings.TrimSpace(string(out)), nil
}
