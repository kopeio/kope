package fi

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"github.com/golang/glog"
)

func ReadersEqual(l, r io.Reader) (bool, error) {
	lBuf := make([]byte, 32 * 1024)
	rBuf := make([]byte, 32 * 1024)

	for {
		nL, err := io.ReadFull(l, lBuf)
		if err != nil && err != io.ErrUnexpectedEOF && err != io.EOF {
			return false, err
		}
		nR, err := io.ReadFull(r, rBuf)
		if err != nil && err != io.ErrUnexpectedEOF && err != io.EOF {
			return false, err
		}

		if nL != nR {
			return false, nil
		}

		if nL == 0 {
			return true, nil
		}

		if nL == len(lBuf) {
			if !bytes.Equal(lBuf, rBuf) {
				return false, nil
			}
		} else {
			if !bytes.Equal(lBuf[:nL], rBuf[:nR]) {
				return false, nil
			}
		}
	}
}

func HasContents(path string, contents []byte) (bool, error) {
	in, err := os.Open(path)
	if err != nil {
		return false, fmt.Errorf("error opening file %q: %v", path, err)
	}
	defer in.Close()

	// TODO: Stream?  But probably not, because we should only be doing this for smallish files
	inContents, err := ioutil.ReadAll(in)
	if err != nil {
		return false, fmt.Errorf("error reading file %q: %v", path, err)
	}

	return bytes.Equal(inContents, contents), nil
}

func SafeClose(r io.Reader) {
	if r == nil {
		return
	}
	closer, ok := r.(io.Closer)
	if !ok {
		return
	}
	err := closer.Close()
	if err != nil {
		glog.Warningf("unexpected error closing stream: %v", err)
	}
}