# cmdgo
A utility for creating command line interfaces that can run in interactive mode, from parameters, from the environment, and from files (json, xml, yaml). The command properties can be validated or populated dynamically.

## Example Code

```go
// myprogram
package main

import (
  "fmt"
  "os"
  cmdgo "github.com/ClickerMonkey/cmdgo/pkg"
)

type Echo struct {
  Message string `prompt:"Enter message" help:"The message to enter" default:"Hello World" min:"2" env:"ECHO_MESSAGE" arg:"msg"`
}

func (echo *Echo) Execute(ctx cmdgo.Context) error {
  fmt.Printf("ECHO: %s\n", echo.Message)
  return nil
}

func main() {
  cmdgo.Register("echo", Echo{})

  ctx := cmdgo.NewContext()
  err := cmdgo.Execute(ctx, os.Args[1:])
  if err != nil {
    panic(err)
  }
}
```

### Example Usage

```
# By default its interactive unless you specify arguments or files to import
> ./myprogram echo
Enter message (Hello World): Hi
ECHO: Hi

> ./myprogram echo --msg Ho
ECHO: Ho

> ./myprogram echo
Enter message (Hello World): help!
The message to enter
Enter message (Hello World): Lets go
ECHO: Lets go

> ECHO_MESSAGE=Hey ./myprogram echo --interactive no
ECHO: Hey

> ./myprogram echo --interactive no
ECHO: Hello World

> ./myprogram echo --msg A
*fails since message is too small*

> ./myprogram echo --json path/to/json/file
# {"Message":"From json!"}
ECHO: From json!

> ./myprogram echo --xml path/to/xml/file
# <Root><Message>From xml!</Message></Root>
ECHO: From xml!

> ./myprogram echo --yaml path/to/yaml/file
# message: From yaml!
ECHO: From yaml!
```
