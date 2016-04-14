package main

import (
	//"github.com/kopeio/kope/cmd"
	"github.com/golang/glog"
	"github.com/kopeio/kope/pkg/loader"
	"flag"
	"github.com/kopeio/kope/pkg/units/gceunits"
	"reflect"
)

func main() {
	//cmd.Execute()

	flag.Parse()

	baseDir := "./pkg/tree/_gce"

	// To avoid problems with golang typing, we have a list of seeds
	// TODO: move to file
	seeds := make(map[string]interface{})
	seeds["nodeCount"] = 2

	ol := loader.NewOptionsLoader(seeds)
	err := ol.WalkDirectory(baseDir)
	if err != nil {
		glog.Exitf("error processing directory %q: %v", baseDir, err)
	}

	options, err := ol.Build()
	if err != nil {
		glog.Exitf("error building options: %v", err)
	}

	l := loader.NewLoader(options)
	l.AddType("persistentDisk", reflect.TypeOf(&gceunits.PersistentDisk{}))
	l.AddType("instance", reflect.TypeOf(&gceunits.Instance{}))
	l.AddType("instanceTemplate", reflect.TypeOf(&gceunits.InstanceTemplate{}))
	l.AddType("network", reflect.TypeOf(&gceunits.Network{}))
	l.AddType("managedInstanceGroup", reflect.TypeOf(&gceunits.ManagedInstanceGroup{}))
	l.AddType("firewallRule", reflect.TypeOf(&gceunits.FirewallRule{}))
	l.AddType("ipAddress", reflect.TypeOf(&gceunits.IPAddress{}))

	err = l.WalkDirectory(baseDir)
	if err != nil {
		glog.Exitf("error processing directory %q: %v", baseDir, err)
	}

	o, err := l.Build()
	if err != nil {
		glog.Exitf("error building objects: %v", err)
	}
	glog.Infof("%v", o)
	glog.Infof("SUCCESS")
}
