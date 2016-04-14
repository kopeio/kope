package kutil

import (
	"fmt"
	"github.com/golang/glog"
	"io/ioutil"
	"net/http"
	"strings"
)

type KubernetesReleases struct {
}

func (c *KubernetesReleases) Latest() (string, error) {
	u := "https://storage.googleapis.com/kubernetes-release/release/stable.txt"

	glog.V(2).Infof("Requesting URL %q", u)
	response, err := http.Get(u)
	if err != nil {
		return "", fmt.Errorf("error getting URL %q: %v", u, err)
	}

	defer response.Body.Close()
	contents, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return "", fmt.Errorf("error reading response from URL %q: %v", u, err)
	}
	v := string(contents)
	v = strings.TrimSpace(v)
	return v, nil
}
