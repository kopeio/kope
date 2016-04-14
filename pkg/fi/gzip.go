package fi

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
)

func GzipBytes(d []byte) ([]byte, error) {
	var out bytes.Buffer
	w := gzip.NewWriter(&out)
	_, err := w.Write(d)
	if err != nil {
		return nil, fmt.Errorf("error compressing data: %v", err)
	}
	err = w.Close()
	if err != nil {
		return nil, fmt.Errorf("error compressing data: %v", err)
	}
	return out.Bytes(), nil
}

func GunzipBytes(d []byte) ([]byte, error) {
	var out bytes.Buffer
	in := bytes.NewReader(d)
	r, err := gzip.NewReader(in)
	if err != nil {
		return nil, fmt.Errorf("error building gunzip reader: %v", err)
	}
	defer r.Close()
	_, err = io.Copy(&out, r)
	if err != nil {
		return nil, fmt.Errorf("error decompressing data: %v", err)
	}
	return out.Bytes(), nil
}
