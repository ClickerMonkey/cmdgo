package cmdgo

import (
	"reflect"
)

type Instance struct {
	Value        reflect.Value
	PropertyMap  map[string]*Property
	PropertyList []*Property
}

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

func GetSubInstance(value any, prop Property) Instance {
	instance := GetInstance(value)

	if concreteKind(instance.Value) != reflect.Struct {
		instance.AddProperty(&Property{
			Value:       instance.Value,
			Type:        instance.Value.Type(),
			Name:        prop.Name,
			PromptText:  prop.PromptText,
			PromptMulti: prop.PromptMulti,
			Options:     prop.Options,
		})
	}

	return instance
}

func (inst *Instance) Capture(ctx *Context, args *[]string) error {
	valueRaw := inst.Value.Interface()

	if dynamic, ok := valueRaw.(Dynamic); ok {
		err := dynamic.Update(ctx, nil, inst)
		if err != nil {
			return err
		}
	}

	for _, property := range inst.PropertyList {
		err := property.Load()
		if err != nil {
			return err
		}

		err = property.FromArgs(ctx, args)
		if err != nil {
			return err
		}

		if ctx.Prompt != nil {
			err = property.Prompt(ctx)
			if err != nil {
				return err
			}
		}

		err = property.Validate()
		if err != nil {
			return err
		}

		if dynamic, ok := valueRaw.(Dynamic); ok {
			err = dynamic.Update(ctx, property, inst)
			if err != nil {
				return err
			}
		}
	}

	if validate, ok := valueRaw.(Validator); ok {
		err := validate.Validate(ctx)
		if err != nil {
			return err
		}
	}

	return nil
}

func (inst Instance) IsDefault() bool {
	for _, prop := range inst.PropertyList {
		if !prop.IsDefault() {
			return false
		}
	}
	return true
}

func (inst Instance) Count(match Match[PropertyFlags]) int {
	count := 0
	for _, prop := range inst.PropertyList {
		if prop.Flags.Is(match) {
			count++
		}
	}
	return count
}

func (inst Instance) Flags() Flags[PropertyFlags] {
	flags := Flags[PropertyFlags]{}
	for _, prop := range inst.PropertyList {
		flags.Set(prop.Flags.value)
	}
	return flags
}

func (inst *Instance) AddProperty(prop *Property) {
	key := Normalize(prop.Name)

	inst.PropertyMap[key] = prop
	inst.PropertyList = append(inst.PropertyList, prop)
}

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
