package cmdgo

import (
	"reflect"
)

type CommandInstance struct {
	Value        reflect.Value
	PropertyMap  map[string]*CommandProperty
	PropertyList []*CommandProperty
}

func GetInstance(value any) CommandInstance {
	reflectValue := reflect.ValueOf(value)
	structValue := ConcreteValue(reflectValue)

	instance := CommandInstance{
		Value:        reflectValue,
		PropertyMap:  make(map[string]*CommandProperty),
		PropertyList: make([]*CommandProperty, 0),
	}

	addProperties(structValue, &instance)

	return instance
}

func (cmd *CommandInstance) Capture(ctx CommandContext, args []string, prompt bool) error {
	valueRaw := cmd.Value.Interface()

	if dynamic, ok := valueRaw.(CommandDynamic); ok {
		err := dynamic.Update(ctx, nil, cmd)
		if err != nil {
			return err
		}
	}

	for _, property := range cmd.PropertyList {
		err := property.Load()
		if err != nil {
			return err
		}

		err = property.FromArgs(ctx, args)
		if err != nil {
			return err
		}

		if prompt {
			err := property.Prompt(ctx)
			if err != nil {
				return err
			}
		}

		err = property.Validate()
		if err != nil {
			return err
		}

		if dynamic, ok := valueRaw.(CommandDynamic); ok {
			err := dynamic.Update(ctx, property, cmd)
			if err != nil {
				return err
			}
		}
	}

	if validate, ok := valueRaw.(CommandValidator); ok {
		err := validate.Validate(ctx)
		if err != nil {
			return err
		}
	}

	return nil
}

func addProperties(structValue reflect.Value, instance *CommandInstance) {
	if structValue.Kind() != reflect.Struct {
		return
	}

	for i := 0; i < structValue.NumField(); i++ {
		fieldValue := structValue.Field(i)
		if !fieldValue.CanSet() {
			continue
		}

		field := structValue.Type().Field(i)

		if field.Anonymous {
			addProperties(fieldValue, instance)
		} else {
			key := Normalize(field.Name)
			property := getCommandProperty(field, fieldValue)

			instance.PropertyMap[key] = &property
			instance.PropertyList = append(instance.PropertyList, &property)
		}
	}
}
