package fi

import (
	"bytes"
	"fmt"
	"strconv"
	"io"
	"strings"
)

type ScriptWriter struct {
	buffer bytes.Buffer
}

func (w *ScriptWriter) SetVar(key string, value string) {
	//	buffer.WriteString(fmt.Sprintf("readonly %s='%s'\n", key, value))
	w.buffer.WriteString(fmt.Sprintf("%s='%s'\n", key, value))
}

func (w *ScriptWriter) SetVarInt(key string, value int) {
	w.SetVar(key, strconv.Itoa(value))
}

func (w *ScriptWriter) SetVarBool(key string, value bool) {
	var v string
	if value {
		v = "true"
	} else {
		v = "false"
	}
	w.SetVar(key, v)
}

func (w *ScriptWriter) WriteString(s string) {
	w.buffer.WriteString(s)
}

func (sw *ScriptWriter) WriteTo(w io.Writer) error {
	_, err := sw.buffer.WriteTo(w)
	return err
}

func (sw *ScriptWriter) AsString() string {
	return sw.buffer.String()
}



//func (s *ScriptWriter) CopyTemplate(key string, replacements map[string]string) {
//	templatePath := path.Join(templateDir, key)
//	contents, err := ioutil.ReadFile(templatePath)
//	if err != nil {
//		glog.Fatalf("error reading template (%s): %v", templatePath, err)
//	}
//
//	for _, line := range strings.Split(string(contents), "\n") {
//		if strings.HasPrefix(line, "#") {
//			continue
//		}
//
//		// This is a workaround to get under the 16KB limit for instance-data,
//		// until we move more functionality into the bootstrap program
//		reString := regexp.QuoteMeta("'$(echo \"$") + "(\\w*)" + regexp.QuoteMeta("\" | sed -e \"s/'/''/g\")'")
//		re, err := regexp.Compile(reString)
//		if err != nil {
//			glog.Fatalf("error compiling regex (%q): %v", reString, err)
//		}
//		if re.MatchString(line) {
//			matches := re.FindStringSubmatch(line)
//			key := matches[1]
//			v, found := replacements[key]
//			if found {
//				newLine := re.ReplaceAllString(line, "'" + v + "'")
//				glog.V(2).Infof("Replace line %q with %q", line, newLine)
//				line = newLine
//			} else {
//				glog.V(2).Infof("key not found: %q", key)
//			}
//		}
//
//		s.WriteString(line + "\n")
//	}
//}

func (s *ScriptWriter) WriteHereDoc(dest string, contents string) {
	s.WriteString("cat << E_O_F > " + dest + "\n")
	for _, line := range strings.Split(contents, "\n") {
		// TODO: Escaping?
		s.WriteString(line + "\n")
	}
	s.WriteString("E_O_F\n\n")
}

