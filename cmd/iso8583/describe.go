package main

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/cimartinez3/iso8583"
	"github.com/cimartinez3/iso8583/cmd/iso8583/describe"
	"github.com/cimartinez3/iso8583/specs"
)

var availableSpecs = map[string]*iso8583.MessageSpec{
	"87ascii": specs.Spec87ASCII,
	"87hex":   specs.Spec87Hex,
}

func describeMessage(paths []string, spec *iso8583.MessageSpec) error {
	for _, path := range paths {
		message, err := createMessageFromFile(path, spec)
		if err != nil {
			if message == nil {
				return fmt.Errorf("creating message from file: %w", err)
			}

			fmt.Fprintf(os.Stdout, "Failed to create message from file: %v\n", err)
			fmt.Fprintf(os.Stdout, "Trying to describe file anyway...\n")
		}

		err = describe.Message(os.Stdout, message)
		if err != nil {
			return fmt.Errorf("describing message: %w", err)
		}
	}
	return nil
}

func Describe(paths []string, specName string) error {
	spec := availableSpecs[specName]
	if spec == nil {
		return fmt.Errorf("unknown built-in spec %s", specName)
	}

	return describeMessage(paths, spec)
}

func DescribeWithSpecFile(paths []string, specFileName string) error {
	spec, err := createSpecFromFile(specFileName)
	if err != nil || spec == nil {
		return fmt.Errorf("creating spec from file: %w", err)
	}

	return describeMessage(paths, spec)
}

func createMessageFromFile(path string, spec *iso8583.MessageSpec) (*iso8583.Message, error) {
	fd, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("opening file %s: %v", path, err)
	}
	defer fd.Close()

	raw, err := ioutil.ReadAll(fd)
	if err != nil {
		return nil, fmt.Errorf("reading file %s: %v", path, err)
	}

	message := iso8583.NewMessage(spec)
	err = message.Unpack(raw)
	if err != nil {
		return message, fmt.Errorf("unpacking message: %v", err)
	}

	return message, nil
}

func createSpecFromFile(path string) (*iso8583.MessageSpec, error) {
	fd, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("opening file %s: %v", path, err)
	}
	defer fd.Close()

	raw, err := ioutil.ReadAll(fd)
	if err != nil {
		return nil, fmt.Errorf("reading file %s: %v", path, err)
	}

	return specs.Builder.ImportJSON(raw)
}
