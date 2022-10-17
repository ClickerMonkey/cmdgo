package main

import (
	"encoding/json"
	"fmt"

	"github.com/ClickerMonkey/cmdgo"
)

type Profile struct {
	Name        string `prompt:"Your name" min:"2"`
	Age         *int   `prompt:"Your age"`
	Password    string `prompt:"Your password" prompt-options:"hidden"`
	FaveNumbers []int  `prompt:"Favorite numbers" prompt-options:"start:-,end:,more:More?" arg:"favenum" min:"3"`
	FaveMovies  []struct {
		Title  string
		Rating float32 `prompt:"Rating (0-10)" min:"0" max:"10"`
	} `prompt-options:"start:Do you have any favorite movies?,end:,more:More?" arg:"movies"`
}

func (prof *Profile) Execute(opts *cmdgo.Options) error {
	result, _ := json.Marshal(prof)
	fmt.Printf("\nProfile: %s\n", result)

	return nil
}

func main() {
	cmdgo.Register(cmdgo.RegistryEntry{
		Name:      "profile",
		Aliases:   []string{""},
		HelpShort: "Gets info about you",
		HelpLong:  "A command which prompts the user for their name, age, password, favorite numbers, and favorite movies.",
		Command:   Profile{},
	})

	opts := cmdgo.NewOptions().Program()
	opts.CaptureExitSignal()

	err := cmdgo.Execute(opts)
	if err != nil {
		panic(err)
	}
}
