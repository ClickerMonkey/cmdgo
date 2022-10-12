# cmdgo
A utility for creating command line interfaces that can run in interactive mode, from parameters, from the environment, and from files (json, xml, yaml). The command properties can be validated or populated dynamically.

## Example Code

```go
// myprogram
package main

import (
  "fmt"
  "github.com/ClickerMonkey/cmdgo"
)

type Echo struct {
  Message string `prompt:"Enter message" help:"The message to enter" default:"Hello World" min:"2" env:"ECHO_MESSAGE" arg:"msg"`
}

func (echo *Echo) Execute(opts cmdgo.Options) error {
  opts.Printf("ECHO: %s\n", echo.Message)
  return nil
}

func main() {
  cmdgo.Register(cmdgo.RegistryEntry{
    Name: "echo", 
    Command: Echo{},
  })

  opts := cmdgo.NewOptions().Program()
  err := cmdgo.Execute(opts)
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
# <Echo><Message>From xml!</Message></Echo>
ECHO: From xml!

> ./myprogram echo --yaml path/to/yaml/file
# message: From yaml!
ECHO: From yaml!
```

### Struct tags
Various struct tags can be used to control how values are pulled from arguments, json, yaml, xml, and prompting the user.

- `json` This tag is utilized normally
  - \`json:"JSON property override,omitempty"`
- `xml` This tag is utilized normally
  - \`xml:"XML element override,omitempty"`
- `yaml` This tag is utilized normally
  - \`yaml:"YAML property override,omitempty"`
- `prompt` The text to display to the user when prompting for a single value. The current/default may be added to this prompt in parenthesis along with ": ".
  - `prompt:"Your name"`
  - `prompt:-` (does not prompt the user for this field)
- `prompt-options` Various options for controlling prompting. They are key:value pairs separated by commas, where value is optional.
  - `start` A message to display when starting a complex value and asking if they want to enter a value for it. If the value is not specified it doesn't prompt the user and just assumes it should start.
  - `more` A message to display when a value has been added to a map or slice and we want to know if more values should be added. The user must enter y to add more.
  - `end` A message to display when the complex value is done being prompted.
  - `multi` The property accepts multiple lines of input and will stop prompting when an empty line is given.
  - `hidden` The property input should be hidden from the user. (ex: passwords)
  - `verify` The user is prompted to re-enter the value to confirm it.
  - `reprompt` The user is repromproted for existing values in the property slice or map. Has no affect for other types.
  - `tries` A maximum number of times to try to get a valid value from the user. This overrides the Context's RepromptOnInvalid.
  - Example: `prompt-options:"start:,end:Thank you for your feedback!,multi,more:Do you have any other questions?"`
- `help` The text to display if the user is prompted for a value and enters "help!" (help text can be changed or disabled on the Context). The prompt will display the help and prompt for a value one more time.
- `default-text` The text to display in place of the current value for a field. If a field contains sensitive data, you can use this to mask it.
- `default` The default value for the field. This is populated on capture assuming no environment variables are found.
- `default-mode` If "hide" then if a field has a current value it won't be displayed when prompting the user.
- `options` A comma delimited list of key:value pairs that are acceptable values. If no values are given the keys are the values. If values are given then the user input is matched to a key and the value is used. Options handle partial keys, so if an option is "hello" and they enter "he" and no other options start with "he" then the value will be the value paired with "hello" or "hello" if there is no value.
  - `options:"a:1,b:2,c:3"` The user can enter a, b, or c and it converts it to the number 1, 2, and 3 respectively.
- `min` The minimum required slice length, map length, string length, or numeric value (inclusive). When prompting for a map or slice it will prompt for this many.
- `max` The maximum allowed slice length, map length, string length, or numeric value (inclusive). When prompting for a map or slice and this length is met capturing will end for the value.
- `env` The environment variables to look for to populate the field.
- `arg` The override for the argument name. By default the argument is the normalized name of the field.
  - `arg:"msg"` (if opts.ArgPrefix is -- then the user can specify this field value with --msg).
  - `arg:"-"` (does not pull value from the arguments)