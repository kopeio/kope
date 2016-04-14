package loader

import (
	"text/template"
	"fmt"
	"bytes"
	"reflect"
	"gopkg.in/yaml.v2"
	"github.com/golang/glog"
	"strings"
	"os"
	"path"
	"io/ioutil"
)

const maxIterations = 10

type OptionsLoader struct {
	seeds     map[string]interface{}
	templates []*template.Template
}

func NewOptionsLoader(seeds map[string]interface{}) *OptionsLoader {
	l := &OptionsLoader{}
	l.seeds = seeds
	return l
}

func (l*OptionsLoader) iterate(inOptions map[string]interface{}) (map[string]interface{}, error) {
	options := make(map[string]interface{})
	for _, t := range l.templates {
		var buffer bytes.Buffer
		err := t.ExecuteTemplate(&buffer, t.Name(), inOptions)
		if err != nil {
			return nil, fmt.Errorf("error executing template %q: %v", t.Name(), err)
		}

		yamlBytes := buffer.Bytes()
		var o map[string]interface{}
		err = yaml.Unmarshal(yamlBytes, &o)
		if err != nil {
			// TODO: It would be nice if yaml returned us the line number here
			glog.Infof("error parsing yaml.  yaml follows:")
			for i, line := range strings.Split(string(yamlBytes), "\n") {
				fmt.Fprintf(os.Stderr, "%3d: %s\n", i, line)
			}
			return nil, fmt.Errorf("error parsing yaml %q: %v", t.Name(), err)
		}

		for k, v := range o {
			options[k] = v
		}
	}

	return options, nil
}

func (l*OptionsLoader) Build() (map[string]interface{}, error) {
	options := make(map[string]interface{})
	for k, v := range l.seeds {
		options[k] = v
	}

	iteration := 0
	for {
		nextOptions, err := l.iterate(options)
		if err != nil {
			return nil, err
		}

		if reflect.DeepEqual(options, nextOptions) {
			for k, v := range options {
				glog.V(2).Infof("Options:  %s=%v", k, v)
			}

			return options, nil
		}

		iteration++
		if iteration > maxIterations {
			for k, v := range options {
				glog.Infof("N:   %s=%v", k, v)
			}
			for k, v := range nextOptions {
				glog.Infof("N+1: %s=%v", k, v)
			}

			return nil, fmt.Errorf("options did not converge after %d iterations", maxIterations)
		}

		options = nextOptions
	}
}

func (l*OptionsLoader) WalkDirectory(baseDir string) error {
	files, err := ioutil.ReadDir(baseDir)
	if err != nil {
		return fmt.Errorf("error reading directory %q: %v", baseDir, err)
	}

	for _, f := range files {
		if !isOptions(f.Name()) {
			continue
		}
		p := path.Join(baseDir, f.Name())
		contents, err := ioutil.ReadFile(p)
		if err != nil {
			return fmt.Errorf("error loading file %q: %v", p, err)
		}

		t := template.New(p)
		_, err = t.Parse(string(contents))
		if err != nil {
			return fmt.Errorf("error parsing template %q: %v", p, err)
		}

		t.Option("missingkey=zero")

		l.templates = append(l.templates, t)
	}

	return nil
}
