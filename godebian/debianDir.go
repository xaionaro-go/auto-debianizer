package godebian

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
)

var (
	ErrInvalidControl error = fmt.Errorf(`Invalid control file (missed ":"?)`)
)

type debianControlSection struct {
	fields map[string]string
}

type debianControl struct {
	sections []debianControlSection
}

func NewDebianControl() (result *debianControl, err error) {
	result = &debianControl{}
	err = result.ParseControlFile()
	return
}

func (control *debianControl) ParseControlFile() error {
	controlRaw, err := ioutil.ReadFile("debian/control")
	if err != nil {
		return err
	}

	controlLines := strings.Split(string(controlRaw), "\n")

	control.sections = []debianControlSection{}
	var prevKey string
	section := make(map[string]string)
	for _, line := range controlLines {
		if line == "" {		// New section
			control.sections = append(control.sections, debianControlSection{fields: section})
			section = make(map[string]string)
			continue
		}

		if line[0:1] == " " {	// The last field is continued
			section[prevKey] += line
			continue
		}

		// New field in this section:

		kvSplitPosition := strings.Index(line, ":")
		if kvSplitPosition == -1 {
			return ErrInvalidControl
		}

		key   := line[0:kvSplitPosition-1]
		var value string
		if len(line) > kvSplitPosition+1 {
			value = line[kvSplitPosition+1:]
		}

		section[key] = value

		prevKey = key
	}
	control.sections = append(control.sections, debianControlSection{fields: section})

	return nil
}

func (control debianControl) Write() error {
	f, err := os.Create("debian/control")
	if err != nil {
		return err
	}
	defer f.Close()

	w := bufio.NewWriter(f)

	for _, section := range control.sections {
		for key, value := range section.fields {
			_, err := w.WriteString(key+": "+value+"\n")
			if err != nil {
				return err
			}
		}
		_, err := w.WriteString("\n")
		if err != nil {
			return err
		}
	}

	w.Flush()
	return nil
}

func (control *debianControl) MainSection() *debianControlSection {
	return &control.sections[0]
}

func (section debianControlSection) Get(key string) string {
	return section.fields[key]
}

func (section *debianControlSection) Set(key string, value string) {
	section.fields[key] = value
}

