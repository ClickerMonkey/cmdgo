package main

import (
	"encoding/json"
	"fmt"

	cmdgo "github.com/ClickerMonkey/cmdgo/pkg"
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

func (prof *Profile) Execute(ctx *cmdgo.Context) error {
	result, _ := json.Marshal(prof)
	fmt.Printf("\nProfile: %s\n", result)

	return nil
}

func main() {
	cmdgo.Register("profile", Profile{})
	cmdgo.Register("", Profile{})

	ctx := cmdgo.NewContext().Program()
	err := cmdgo.Execute(ctx)
	if err != nil {
		panic(err)
	}
}
