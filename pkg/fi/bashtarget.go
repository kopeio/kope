package fi

import (
	"bytes"
	"fmt"
	"io"
	"path"
	"strconv"
	"strings"

	"github.com/golang/glog"
	"os"
)

type BashTarget struct {
	// TODO: Remove cloud
	Cloud                *AWSCloud
	filestore            FileStore
	commands             []*BashCommand
	ec2Args              []string
	autoscalingArgs      []string
	iamArgs              []string
	vars                 map[string]*BashVar
	prefixCounts         map[string]int
	resourcePrefixCounts map[string]int
}

var _ Target = &BashTarget{}

func NewBashTarget(cloud *AWSCloud, filestore FileStore) *BashTarget {
	b := &BashTarget{Cloud: cloud, filestore: filestore}
	b.ec2Args = []string{"aws", "ec2"}
	b.autoscalingArgs = []string{"aws", "autoscaling"}
	b.iamArgs = []string{"aws", "iam"}
	b.vars = make(map[string]*BashVar)
	b.prefixCounts = make(map[string]int)
	b.resourcePrefixCounts = make(map[string]int)
	return b
}

type BashVar struct {
	name        string
	staticValue *string
}

func getVariablePrefix(v Unit) string {
	name := GetTypeName(v)
	name = strings.ToUpper(name)
	return name
}

func getKey(v Unit) string {
	return v.Path()
}

func (t *BashTarget) CreateVar(v Unit) *BashVar {
	key := getKey(v)
	bv, found := t.vars[key]
	if found {
		glog.Fatalf("Attempt to create variable twice for %q: %v", key, v)
	}
	bv = &BashVar{}
	prefix := getVariablePrefix(v)
	n := t.prefixCounts[prefix]
	n++
	t.prefixCounts[prefix] = n

	bv.name = prefix + "_" + strconv.Itoa(n)
	t.vars[key] = bv
	return bv
}

type BashCommand struct {
	parent   *BashTarget
	args     []string
	assignTo string
}

func (c *BashCommand) AssignTo(s Unit) *BashCommand {
	return c.AssignToSuffixedVariable(s, "")
}

func (c *BashCommand) AssignToSuffixedVariable(s Unit, suffix string) *BashCommand {
	bv := c.parent.vars[getKey(s)]
	if bv == nil {
		glog.Fatal("no variable assigned to ", s)
	}
	if bv.name == "" {
		glog.Fatal("no name for bash var assignment")
	}
	c.assignTo = bv.name + suffix
	return c
}

func (c *BashCommand) String() string {
	if c.assignTo != "" {
		return c.assignTo + "=`" + strings.Join(c.args, " ") + "`"
	} else {
		return strings.Join(c.args, " ")
	}
}

func (c *BashCommand) PrintShellCommand(w io.Writer) error {
	var buf bytes.Buffer

	if c.assignTo != "" {
		buf.WriteString(c.assignTo)
		buf.WriteString("=`")
	}

	for i, arg := range c.args {
		if i != 0 {
			buf.WriteString(" ")
		}
		buf.WriteString(arg)
	}

	if c.assignTo != "" {
		buf.WriteString("`")
	}

	buf.WriteString("\n")

	_, err := buf.WriteTo(w)
	return err
}

func (t *BashTarget) ReadVar(s Unit) string {
	return t.ReadVarWithSuffix(s, "")
}

func (t *BashTarget) ReadVarWithSuffix(s Unit, suffix string) string {
	bv := t.vars[getKey(s)]
	if bv == nil {
		glog.Fatal("no variable assigned to ", s)
	}
	// TODO: Escaping?
	return "${" + bv.name + suffix + "}"
}

func (t *BashTarget) DebugDump() {
	for _, cmd := range t.commands {
		glog.Info("CMD: ", cmd)
	}
}

func (t *BashTarget) PrintShellCommands(w io.Writer) error {
	var header bytes.Buffer
	header.WriteString("#!/bin/bash\n")
	header.WriteString("set -ex\n\n")
	header.WriteString(". ./helpers\n\n")

	for k, v := range t.Cloud.EnvVars() {
		header.WriteString("export " + k + "=" + BashQuoteString(v) + "\n")
	}

	_, err := header.WriteTo(w)
	if err != nil {
		return err
	}

	for _, cmd := range t.commands {
		err = cmd.PrintShellCommand(w)
		if err != nil {
			return err
		}
	}
	return nil
}

func (t *BashTarget) AddEC2Command(args ...string) *BashCommand {
	cmd := &BashCommand{parent: t}
	cmd.args = t.ec2Args
	cmd.args = append(cmd.args, args...)

	return t.AddCommand(cmd)
}

func (t *BashTarget) AddBashCommand(args ...string) *BashCommand {
	cmd := &BashCommand{parent: t}
	cmd.args = args

	return t.AddCommand(cmd)
}

func (t *BashTarget) AddAutoscalingCommand(args ...string) *BashCommand {
	cmd := &BashCommand{parent: t}
	cmd.args = t.autoscalingArgs
	cmd.args = append(cmd.args, args...)

	return t.AddCommand(cmd)
}

func (t *BashTarget) AddS3Command(region string, args ...string) *BashCommand {
	cmd := &BashCommand{parent: t}
	cmd.args = []string{"aws", "s3", "--region", region}
	cmd.args = append(cmd.args, args...)

	return t.AddCommand(cmd)
}

func (t *BashTarget) AddS3APICommand(region string, args ...string) *BashCommand {
	cmd := &BashCommand{parent: t}
	cmd.args = []string{"aws", "s3api", "--region", region}
	cmd.args = append(cmd.args, args...)

	return t.AddCommand(cmd)
}

func (t *BashTarget) AddIAMCommand(args ...string) *BashCommand {
	cmd := &BashCommand{parent: t}
	cmd.args = t.iamArgs
	cmd.args = append(cmd.args, args...)

	return t.AddCommand(cmd)
}

func BashQuoteString(s string) string {
	// TODO: Escaping
	var quoted bytes.Buffer
	for _, c := range s {
		switch c {
		case '"':
			quoted.WriteString("\\\"")
		default:
			quoted.WriteString(string(c))
		}
	}

	return "\"" + string(quoted.Bytes()) + "\""
}

func (t *BashTarget) AddAWSTags(s Unit, expected map[string]string) error {
	resourceId, exists := t.FindValue(s)
	var missing map[string]string
	if exists {
		actual, err := t.Cloud.GetTags(resourceId)
		if err != nil {
			return fmt.Errorf("unexpected error fetching tags for resource: %v", err)
		}

		missing = map[string]string{}
		for k, v := range expected {
			actualValue, found := actual[k]
			if found && actualValue == v {
				continue
			}
			missing[k] = v
		}
	} else {
		missing = expected
	}

	for name, value := range missing {
		cmd := &BashCommand{}
		cmd.args = []string{"add-tag", t.ReadVar(s), BashQuoteString(name), BashQuoteString(value)}
		t.AddCommand(cmd)
	}

	return nil
}

func (t *BashTarget) AddCommand(cmd *BashCommand) *BashCommand {
	cmd.parent = t
	glog.V(2).Infof("Add bash command: %v", cmd)
	t.commands = append(t.commands, cmd)

	return cmd
}

func (t *BashTarget) AddAssignment(u Unit, value string) {
	bv := t.vars[getKey(u)]
	if bv == nil {
		glog.Fatal("no variable assigned to ", u)
	}

	cmd := &BashCommand{}
	assign := bv.name + "=" + BashQuoteString(value)
	cmd.args = []string{assign}
	t.AddCommand(cmd)

	bv.staticValue = &value
}

func (t *BashTarget) FindValue(u Unit) (string, bool) {
	bv := t.vars[getKey(u)]
	if bv == nil {
		glog.Fatal("no variable assigned to ", u)
	}

	if bv.staticValue == nil {
		return "", false
	}
	return *bv.staticValue, true
}

func (t *BashTarget) generateDynamicPath(prefix string) string {
	basePath := "resources"
	n := t.resourcePrefixCounts[prefix]
	n++
	t.resourcePrefixCounts[prefix] = n

	name := prefix + "_" + strconv.Itoa(n)
	p := path.Join(basePath, name)
	return p
}

func (t *BashTarget) AddLocalResource(r Resource) (string, error) {
	switch r := r.(type) {
	case *FileResource:
		return r.Path, nil
	}

	path := t.generateDynamicPath(GetTypeName(r))
	f, err := os.Create(path)
	if err != nil {
		return "", err
	}
	defer func() {
		err := f.Close()
		if err != nil {
			glog.Warning("Error closing resource file", err)
		}
	}()

	err = CopyResource(f, r)
	if err != nil {
		return "", fmt.Errorf("error writing resource: %v", err)
	}

	return path, nil
}

func (t *BashTarget) PutResource(key string, r Resource, hashAlgorithm HashAlgorithm) (string, string, error) {
	if r == nil {
		glog.Fatalf("Attempt to put null resource for %q", key)
	}
	return t.filestore.PutResource(key, r, hashAlgorithm)
}

func (t *BashTarget) WaitForInstanceRunning(instance Unit)  {
	instanceID := t.ReadVar(instance)
	t.AddBashCommand("wait-for-instance-state", instanceID, "running")
}


