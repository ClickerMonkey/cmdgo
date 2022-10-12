package cmdgo

import (
	"reflect"
)

// An instance of a struct and all the properties on it.
type Instance struct {
	Value        reflect.Value
	PropertyMap  map[string]*Property
	PropertyList []*Property
}

// Creates an instance given a value.
func GetInstance(value any) Instance {
	reflected := reflectValue(value)
	concrete := concreteValue(reflected)

	instance := Instance{
		Value:        reflected,
		PropertyMap:  make(map[string]*Property),
		PropertyList: make([]*Property, 0),
	}

	addProperties(concrete, &instance)

	return instance
}

// Creates an instance that is appropriate for the given property.
func GetSubInstance(value any, prop Property) Instance {
	instance := GetInstance(value)

	if concreteKind(instance.Value) != reflect.Struct {
		instance.AddProperty(&Property{
			Value:       instance.Value,
			Type:        instance.Value.Type(),
			Name:        prop.Name,
			PromptText:  prop.PromptText,
			PromptMulti: prop.PromptMulti,
			Choices:     prop.Choices,
		})
	}

	return instance
}

// Capture populates the properties of the instance from arguments and prompting the options.
func (inst *Instance) Capture(opts *Options) error {
	valueRaw := inst.Value.Interface()

	if dynamic, ok := valueRaw.(Dynamic); ok {
		err := dynamic.Update(opts, nil, inst)
		if err != nil {
			return err
		}
	}

	for _, property := range inst.PropertyList {
		err := property.Load()
		if err != nil {
			return err
		}

		err = property.FromArgs(opts)
		if err != nil {
			return err
		}

		err = property.Prompt(opts)
		if err != nil {
			return err
		}

		err = property.Validate()
		if err != nil {
			return err
		}

		if dynamic, ok := valueRaw.(Dynamic); ok {
			err = dynamic.Update(opts, property, inst)
			if err != nil {
				return err
			}
		}
	}

	if validate, ok := valueRaw.(Validator); ok {
		err := validate.Validate(opts)
		if err != nil {
			return err
		}
	}

	return nil
}

// Returns if the value in this instance has all default values.
func (inst Instance) IsDefault() bool {
	for _, prop := range inst.PropertyList {
		if !prop.IsDefault() {
			return false
		}
	}
	return true
}

// Counts how many properties in this instance match.
func (inst Instance) Count(match Match[PropertyFlags]) int {
	count := 0
	for _, prop := range inst.PropertyList {
		if prop.Flags.Is(match) {
			count++
		}
	}
	return count
}

// Builds a set of all flags in all properties in this instance.
func (inst Instance) Flags() Flags[PropertyFlags] {
	flags := Flags[PropertyFlags]{}
	for _, prop := range inst.PropertyList {
		flags.Set(prop.Flags.value)
	}
	return flags
}

// Adds a property to the instance.
func (inst *Instance) AddProperty(prop *Property) {
	key := Normalize(prop.Name)

	inst.PropertyMap[key] = prop
	inst.PropertyList = append(inst.PropertyList, prop)
}

// Adds the properties defined in the struct value to the given instance.
func addProperties(structValue reflect.Value, instance *Instance) {
	if structValue.Kind() != reflect.Struct {
		return
	}

	structType := structValue.Type()

	for i := 0; i < structValue.NumField(); i++ {
		fieldValue := structValue.Field(i)
		if !fieldValue.CanSet() {
			continue
		}

		field := structType.Field(i)

		if field.Anonymous {
			addProperties(fieldValue, instance)
		} else {
			property := getStructProperty(field, fieldValue)
			instance.AddProperty(&property)
		}
	}
}
