package cmdgo

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"strconv"

	yaml "gopkg.in/yaml.v2"
)

func Run(ctx CommandContext, args []string) error {
	cmd, err := Capture(ctx, args)
	if err != nil {
		return err
	}

	err = cmd.Execute(ctx)
	if err != nil {
		return err
	}

	return nil
}

func Capture(ctx CommandContext, args []string) (Command, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("No command given.")
	}

	commandName := args[0]
	command := Get(commandName)

	if command == nil {
		return nil, fmt.Errorf("Command not found: %v", commandName)
	}

	args = args[1:]

	interactiveDefault := "false"
	if len(args) == 0 {
		interactiveDefault = "true"
	}

	interactive, _ := strconv.ParseBool(GetArg("interactive", interactiveDefault, args, ctx.ArgPrefix, true))
	jsonPath := GetArg("json", "", args, ctx.ArgPrefix, false)
	xmlPath := GetArg("xml", "", args, ctx.ArgPrefix, false)
	yamlPath := GetArg("yaml", "", args, ctx.ArgPrefix, false)

	commandInstance := GetInstance(command)

	if jsonPath != "" {
		jsonFile, err := ioutil.ReadFile(jsonPath)
		if err != nil {
			return nil, err
		}
		err = json.Unmarshal(jsonFile, &command)
		if err != nil {
			return nil, err
		}
	}
	if xmlPath != "" {
		xmlFile, err := ioutil.ReadFile(xmlPath)
		if err != nil {
			return nil, err
		}
		err = xml.Unmarshal(xmlFile, &command)
		if err != nil {
			return nil, err
		}
	}
	if yamlPath != "" {
		yamlFile, err := ioutil.ReadFile(yamlPath)
		if err != nil {
			return nil, err
		}
		err = yaml.Unmarshal(yamlFile, &command)
		if err != nil {
			return nil, err
		}
	}

	err := commandInstance.Capture(ctx, args, interactive)
	if err != nil {
		return nil, err
	}

	return command, nil
}
