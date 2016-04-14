package fi

import (
	"bytes"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"hash"
	"io"
	"os"
)

type Resources interface {
	Get(key string) (Resource, bool)
}

type Resource interface {
	Open() (io.ReadSeeker, error)
	//WriteTo(io.Writer) error
	//SameContents(path string) (bool, error)
}

func HashForResource(r Resource, hashAlgorithm HashAlgorithm) (string, error) {
	hasher := NewHasher(hashAlgorithm)
	err := CopyResource(hasher, r)
	if err != nil {
		return "", fmt.Errorf("error while hashing resource: %v", err)
	}
	return hex.EncodeToString(hasher.Sum(nil)), nil
}

func HashesForResource(r Resource, hashAlgorithms []HashAlgorithm) (map[HashAlgorithm]string, error) {
	hashers := make(map[HashAlgorithm]hash.Hash)
	var writers []io.Writer
	for _, hashAlgorithm := range hashAlgorithms {
		if hashers[hashAlgorithm] != nil {
			continue
		}
		hasher := NewHasher(hashAlgorithm)
		hashers[hashAlgorithm] = hasher
		writers = append(writers, hasher)
	}

	w := io.MultiWriter(writers...)

	err := CopyResource(w, r)
	if err != nil {
		return nil, fmt.Errorf("error while hashing resource: %v", err)
	}

	hashes := make(map[HashAlgorithm]string)
	for k, hasher := range hashers {
		hashes[k] = hex.EncodeToString(hasher.Sum(nil))
	}

	return hashes, nil
}

func ResourcesMatch(a, b Resource) (bool, error) {
	aReader, err := a.Open()
	if err != nil {
		return false, err
	}
	defer SafeClose(aReader)

	bReader, err := b.Open()
	if err != nil {
		return false, err
	}
	defer SafeClose(bReader)

	const size = 8192
	aData := make([]byte, size)
	bData := make([]byte, size)

	for {
		aN, aErr := io.ReadFull(aReader, aData)
		if aErr != nil && aErr != io.EOF && aErr != io.ErrUnexpectedEOF {
			return false, aErr
		}

		bN, bErr := io.ReadFull(bReader, bData)
		if bErr != nil && bErr != io.EOF && bErr != io.ErrUnexpectedEOF {
			return false, bErr
		}

		if aErr == nil && bErr == nil {
			if aN != size || bN != size {
				panic("violation of io.ReadFull contract")
			}
			if !bytes.Equal(aData, bData) {
				return false, nil
			}
			continue
		}

		if aN != bN {
			return false, nil
		}

		return bytes.Equal(aData[0:aN], bData[0:bN]), nil
	}
}

func CopyResource(dest io.Writer, r Resource) error {
	in, err := r.Open()
	if err != nil {
		return fmt.Errorf("error opening resource: %v", err)
	}
	defer SafeClose(in)

	_, err = io.Copy(dest, in)
	if err != nil {
		return fmt.Errorf("error copying resource: %v", err)
	}
	return nil
}

func ResourceAsString(r Resource) (string, error) {
	buf := new(bytes.Buffer)
	err := CopyResource(buf, r)
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}

func ResourceAsBase64String(r Resource) (string, error) {
	data, err := ResourceAsBytes(r)
	if err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(data), nil
}

func ResourceAsBytes(r Resource) ([]byte, error) {
	buf := new(bytes.Buffer)
	err := CopyResource(buf, r)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

type ResourcesList struct {
	resources []Resources
}

var _ Resources = &ResourcesList{}

func (l *ResourcesList) Get(key string) (Resource, bool) {
	for _, r := range l.resources {
		resource, found := r.Get(key)
		if found {
			return resource, true
		}
	}
	return nil, false
}

func (l *ResourcesList) Add(r Resources) {
	l.resources = append(l.resources, r)
}

type StringResource struct {
	s string
}

var _ Resource = &StringResource{}

func NewStringResource(s string) *StringResource {
	return &StringResource{s: s}
}

func (s *StringResource) Open() (io.ReadSeeker, error) {
	r := bytes.NewReader([]byte(s.s))
	return r, nil
}

func (s *StringResource) WriteTo(out io.Writer) error {
	_, err := out.Write([]byte(s.s))
	return err
}

func (s *StringResource) SameContents(path string) (bool, error) {
	return HasContents(path, []byte(s.s))
}

type BytesResource struct {
	data []byte
}

var _ Resource = &BytesResource{}

func NewBytesResource(data []byte) *BytesResource {
	return &BytesResource{data: data}
}

func (r *BytesResource) Open() (io.ReadSeeker, error) {
	reader := bytes.NewReader([]byte(r.data))
	return reader, nil
}

type FuncResource struct {
	fn func() ([]byte, error)
}

var _ Resource = &FuncResource{}

func NewFuncResource(fn func() ([]byte, error)) *FuncResource {
	return &FuncResource{fn: fn}
}

func (r *FuncResource) Open() (io.ReadSeeker, error) {
	data, err := r.fn()
	if err != nil {
		return nil, err
	}
	reader := bytes.NewReader(data)
	return reader, nil
}

type FileResource struct {
	Path string
}

var _ Resource = &FileResource{}

func NewFileResource(path string) *FileResource {
	return &FileResource{Path: path}
}
func (r *FileResource) Open() (io.ReadSeeker, error) {
	in, err := os.Open(r.Path)
	if err != nil {
		return nil, fmt.Errorf("error opening file %q: %v", r.Path, err)
	}
	return in, err
}

func (r *FileResource) WriteTo(out io.Writer) error {
	in, err := r.Open()
	defer SafeClose(in)
	_, err = io.Copy(out, in)
	if err != nil {
		return fmt.Errorf("error copying file %q: %v", r.Path, err)
	}
	return err
}
