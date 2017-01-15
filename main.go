package main

import (
	"fmt"
	"github.com/xaionaro-go/auto-debianizer/godebian"
	"os"
	"os/exec"
	"strings"
)

var (
	ErrCannotInstall error = fmt.Errorf("Cannot install the package")
	ErrCannotFixDependencies error = fmt.Errorf("Don't know how to fix depedencies")
)

func installPackage(pkg string) error {
	fmt.Println(`Installing package "`, pkg, `":`)
	out, err := exec.Command("sudo", "apt-get", "install", "-y", pkg).Output()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Got error from \"sudo apt-get install -y "+pkg+"\": ", err.Error(), ": ", string(out))
		return err
	}
	outStr := string(out)
	fmt.Println(outStr)

	if strings.Index(outStr, "") == -1 {
		return ErrCannotInstall
	}

	return nil
}

func debianControlGet(searchKey string) string {
	control, err := godebian.NewDebianControl()
	if err != nil {
		panic(err)
	}
	return control.MainSection().Get(searchKey)

}

func getBuildDependencies() (buildDeps []string, err error) {
	buildDepsUntrimmed := strings.Split(debianControlGet("Build-Depends"), ",")
	for _, buildDepUntrimmed := range buildDepsUntrimmed {
		buildDeps = append(buildDeps, strings.Trim(buildDepUntrimmed, " "))
	}

	return
}
func setBuildDependencies(buildDeps []string) error {
	control, err := godebian.NewDebianControl()
	if err != nil {
		return err
	}

	control.MainSection().Set("Build-Depends", strings.Join(buildDeps, ", "))

	return control.Write()
}

func addBuildDependencyToControl(pkg string) error {
	buildDeps, err := getBuildDependencies()
	if err != nil {
		return err
	}
	return setBuildDependencies(append(buildDeps, pkg))
}

func addBuildDependency(pkg string) error {
	fmt.Printf("Adding package \"%v\" to Build-Depends\n", pkg)

	err := installPackage(pkg)
	if err != nil {
		return err
	}

	return addBuildDependencyToControl(pkg)
}

func fixDependenciesError(buildingOutput string) error {
	fmt.Printf("A dependencies-problem. Trying to fix.\n")

	lines := strings.Split(buildingOutput, "\n")

	var prevLine string
	for _, line := range lines {
		if strings.Index(line, " result: ") != -1 {
			continue
		}
		if strings.Index(line, "error: Requires") == -1 {
			prevLine = line
			continue
		}

		fmt.Printf("A dependencies-problem. The line: %v\n", prevLine)

		words := strings.Split(prevLine, " ")
		packageCategory := strings.Trim(words[2], " ")
		packageName := strings.Trim(words[4], ". ")

		var packageFullname string
		switch packageCategory {
		default:
			packageFullname = packageCategory+"-"+packageName
		}

		return addBuildDependency(packageFullname)
	}

	return ErrCannotFixDependencies
}

func main() {
	try := 0

	for {
		fmt.Printf("A try #%v\n", try)
		try += 1
		_, err := exec.Command("dpkg-buildpackage", "-rfakeroot").Output()
		if err == nil {
			return
		}
		//_, err := exec.Command(`awk`, `BEGIN {theLine=""} {if ($2=="error:"){theLine=$0}} END {print theLine}`, `config.log`).Output()
		out, err := exec.Command(`cat`, `config.log`).Output()
		if err != nil {
			fmt.Fprintln(os.Stderr, "There's no \"config.log\" file :(: %v", err.Error())
			return
		}
		outStr := string(out)
		fmt.Println(outStr)

		depErrorPosition := strings.Index(outStr, "error: Requires")
		if depErrorPosition != -1 {
			err := fixDependenciesError(outStr)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Cannot fix a dependency\n")
				break
			}
			continue
		}

		fmt.Fprintln(os.Stderr, "Don't know what to do :(")
		break
	}
}

