package daemon

import (
	"os"
)

// Created for testing purposes
var fs fileSystem = osFS{}

type fileSystem interface {
	Stat(name string) (os.FileInfo, error)
	Readlink(name string) (string, error)
	Create(name string) (*os.File, error)
	Remove(name string) error
}

// osFS implements fileSystem using the local disk.
type osFS struct{}

func (osFS) Stat(name string) (os.FileInfo, error) { return os.Stat(name) }
func (osFS) Readlink(name string) (string, error)  { return os.Readlink(name) }
func (osFS) Create(name string) (*os.File, error)  { return os.Create(name) }
func (osFS) Remove(name string) error              { return os.Remove(name) }
