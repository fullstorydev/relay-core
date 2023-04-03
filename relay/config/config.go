package config

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

// File exposes an in-memory representation of a configuration file. Files are
// composed of a collection of named Section objects. Each Section contains a
// collection of values of arbitrary type. Generally each Section is associated
// with a plugin or subsystem, and each value corresponds to a configuration
// option.
type File struct {
	sections map[string]*Section
}

// NewFile returns a new, empty File.
func NewFile() *File {
	return &File{
		sections: map[string]*Section{},
	}
}

// NewFileFromYamlString returns a File generated from YAML. Each top-level
// YAML property becomes a Section; the properties it contains become values in
// that Section.
func NewFileFromYamlString(fileYaml string) (*File, error) {
	var yamlSections map[string]map[string]yaml.Node
	if err := yaml.Unmarshal([]byte(fileYaml), &yamlSections); err != nil {
		return nil, err
	}

	file := NewFile()
	for sectionName, sectionValues := range yamlSections {
		section := file.GetOrAddSection(sectionName)
		for valueName, value := range sectionValues {
			section.Set(valueName, value)
		}
	}

	return file, nil
}

// GetOrAddSection returns the Section with the specified name, if one exists.
// If there is no existing Section with that name, an empty Section is created,
// added to the File, and returned.
func (file *File) GetOrAddSection(name string) *Section {
	if file.sections[name] == nil {
		file.sections[name] = NewSection(name)
	}
	return file.sections[name]
}

// LookupOptionalSection returns the section with the specified name, if one
// exists. If not, it returns nil.
func (file *File) LookupOptionalSection(name string) *Section {
	return file.sections[name]
}

// LookupRequiredSection returns the section with the specified, if one exists.
// If not, it returns an error.
func (file *File) LookupRequiredSection(name string) (*Section, error) {
	if file.sections[name] == nil {
		return nil, fmt.Errorf(`Missing required configuration section "%v"`, name)
	}
	return file.sections[name], nil
}

// Section is a named collection of values usually found within a File.
// Generally a Section is associated with a plugin or subsystem, and the values
// it contains represent configuration options for that plugin or subsystem.
type Section struct {
	Name   string
	values map[string]interface{}
}

// NewSection returns a new, empty Section.
func NewSection(name string) *Section {
	return &Section{
		Name:   name,
		values: map[string]interface{}{},
	}
}

// Set adds the provided value to this Section, storing it under the provided
// key. Any type of value may be provided. If the value is a yaml.Node, it will
// automatically be unmarshaled into an appropriate runtime type when a lookup
// occurs.
func (section *Section) Set(key string, value interface{}) {
	section.values[key] = value
}

// lookupValueInSection is an internal helper that attempts to read the value
// with the provided key from the provided Section. If the value has type T, the
// value is returned. If the value has type yaml.Node and can be unmarshaled
// into a T, the unmarshaled value is returned. Otherwise, an error is reported.
func lookupValueInSection[T any](section *Section, key string) (*T, error) {
	nodeOrValue, ok := section.values[key]
	if !ok {
		return nil, nil
	}

	switch typedNodeOrValue := nodeOrValue.(type) {
	case yaml.Node:
		// Detect a value which is completely empty in the YAML source, like
		// "foo"'s value here:
		//   foo:
		// We treat these values as if the key they're associated with ("foo" in
		// this case) is not present. They'd otherwise be treated as the empty
		// string, but that would often lead us to generate error messages which
		// aren't as nice, especially given that we provide default values via
		// environment variables for many configuration options, so that the
		// options are always "present". Empty strings can still be used in the
		// configuration file by surrounding them with explicit quotes.
		if typedNodeOrValue.Kind == yaml.ScalarNode &&
			typedNodeOrValue.Style == 0 &&
			typedNodeOrValue.Value == "" {
			return nil, nil
		}

		var value T
		if err := typedNodeOrValue.Decode(&value); err != nil {
			return nil, err
		}
		return &value, nil

	case T:
		return &typedNodeOrValue, nil

	default:
		return nil, fmt.Errorf(`Found value "%v" of unexpected type`, typedNodeOrValue)
	}
}

// LookupOptional returns the value associated with the provided key, if it's
// present with type T. If it's not present, nil is returned. If it's present
// but has the wrong type, an error is returned.
func LookupOptional[T any](section *Section, key string) (*T, error) {
	if value, err := lookupValueInSection[T](section, key); err != nil {
		return nil, fmt.Errorf(`Invalid value for configuration option "%v" in section "%v": %v`, key, section.Name, err)
	} else {
		return value, nil
	}
}

// LookupRequired returns the value associated with the provided key, if it's
// present with type T. If it's not present, or if it's present but has the
// wrong type, an error is returned.
func LookupRequired[T any](section *Section, key string) (T, error) {
	value, err := LookupOptional[T](section, key)
	if err != nil {
		var zeroValue T
		return zeroValue, err
	}
	if value == nil {
		var zeroValue T
		return zeroValue, fmt.Errorf(`Missing required configuration option "%v" in section "%v"`, key, section.Name)
	}
	return *value, nil
}

// ParseOptional invokes a callback with the value of the provided key, if it's
// present and has type T, and propagates any error the callback returns. If the
// key is not found, the callback is not invoked and no error is reported. If
// it's present and has the wrong type, an error is returned.
func ParseOptional[T any](
	section *Section,
	key string,
	action func(key string, value T) error,
) error {
	value, err := LookupOptional[T](section, key)
	if err != nil {
		return err
	}
	if value == nil {
		return nil
	}

	if err := action(key, *value); err != nil {
		return fmt.Errorf(`Error parsing configuration option "%v" in section "%v": %v`, key, section.Name, err)
	}

	return nil
}

// ParseRequired invokes a callback with the value of the provided key, if it's
// present and has type T, and propagates any error the callback returns. If the
// key is not found, or if it has the wrong type, an error is reported.
func ParseRequired[T any](
	section *Section,
	key string,
	action func(key string, value T) error,
) error {
	value, err := LookupRequired[T](section, key)
	if err != nil {
		return err
	}

	if err := action(key, value); err != nil {
		return fmt.Errorf(`Error parsing configuration option "%v" in section "%v": %v`, key, section.Name, err)
	}

	return nil
}
