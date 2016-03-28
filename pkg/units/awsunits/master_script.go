package awsunits

import (
	"io"

	"github.com/kopeio/kope/pkg/fi"
	"bytes"
)

type MasterScript struct {
	fi.SimpleUnit

	Construct func(c*fi.RunContext) (string, error)
	contents  string
}

func (s *MasterScript) Key() string {
	return "master-script"
}

var _ fi.Resource = &MasterScript{}

type NodeScript struct {
	fi.SimpleUnit

	Construct func(c*fi.RunContext) (string, error)
	contents  string
}

var _ fi.Resource = &NodeScript{}

func (s *NodeScript) Key() string {
	return "node-script"
}

//func (m *NodeScript) Prefix() string {
//	return "node_script"
//}

func (m*MasterScript) Run(c *fi.RunContext) error {
	contents, err := m.Construct(c)
	if err != nil {
		return err
	}
	m.contents = contents
	return nil
}

func (m *MasterScript) Open() (io.ReadSeeker, error) {
	if m.contents == "" {
		panic("executed out of sequence")
	}
	return bytes.NewReader([]byte(m.contents)), nil
}

func (m*NodeScript) Run(c *fi.RunContext) error {
	contents, err := m.Construct(c)
	if err != nil {
		return err
	}
	m.contents = contents
	return nil
}

func (m *NodeScript) Open() (io.ReadSeeker, error) {
	if m.contents == "" {
		panic("executed out of sequence")
	}
	return bytes.NewReader([]byte(m.contents)), nil
}

//func (m *MasterScript) Prefix() string {
//	return "master_script"
//}
