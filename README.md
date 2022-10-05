# cmdgo
A utility for creating command line interfaces that can run in interactive mode, from parameters, from the environment, and from files (json, xml, yaml). The command properties can be validated or populated dynamically.

## Example Code

```go
// myprogram
package main

import (
  "fmt"
  cmdgo "github.com/ClickerMonkey/cmdgo/pkg"
)

type Echo struct {
  Message string `prompt:"Enter message" help:"The message to enter" default:"Hello World" min:"2" env:"ECHO_MESSAGE" arg:"msg"`
}

func (echo *Echo) Execute(ctx cmdgo.CommandContext) error {
  fmt.Printf("ECHO: %s\n", echo.Message)
  return nil
}

func main() {
  cmdgo.Register("echo", func() cmdgo.Command { return &Echo{} })

  ctx := NewStandardContext(map[string]any{})
  err := cmdgo.Run(ctx, os.Args[1:])
  if err != nil {
    panic(err)
  }
}
```

### Example Usage

```
> ./myprogram echo
Enter message: Hi
ECHO: Hi

> ./myprogram echo --msg Ho
ECHO: Ho

> ./myprogram echo
Enter message: help!
The message to enter
Enter message: Lets go
ECHO: Lets go

> ECHO_MESSAGE=Hey ./myprogram echo --interactive false
ECHO: Hey

> ./myprogram echo --interactive false
ECHO: Hello World

> ./myprogram echo --msg A
*fails since message is too small*

> ./myprogram echo --json path/to/json/file
# {"Message":"From json!"}
ECHO: From json!

> ./myprogram echo --xml path/to/xml/file
# <Root><Message>From xml!</Message</Root>
ECHO: From xml!

> ./myprogram echo --yaml path/to/yaml/file
# Message: From yaml!
ECHO: From yaml!
```