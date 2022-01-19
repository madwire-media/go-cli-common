# Go CLI Common
This repo contains shared code across Madwire's development CLI tooling,
including in the [secrets-cli](https://github.com/madwire-media/secrets-cli)

## Examples

### Interactive CLI helpers
```go
package main

import (
    fmt
    os

    clicommon "github.com/madwire-media/go-cli-common"
)

func main() {
    if !clicommon.CliQuestionYesNo("Are you at least 13 years old?") {
        fmt.Println("You need to be 13 years or older to sign up")
        os.Exit(1)
    }

    username := clicommon.CliQuestion("Username")
    password := clicommon.CliQuestionHidden("Password")

    fmt.Println("Welcome!")
}
```

```
$ ./example
Are you at least 13 years old? (y/n): n
You need to be 13 years or older to sign up
$ ./example
Are you at least 13 years old? (y/n): y
Username: foo
Password (hidden):
Welcome!
```

### Tiny privilege escalation framework
```go
package main

import (
    fmt

    clicommon "github.com/madwire-media/go-cli-common"
)

func main() {
    // Handle a superuser action
    clicommon.TryHandleSudo()

    // Re-execute this binary with superuser permissions, and run the
    // DummyAction to log "Hello Github!"
    clicommon.CallSudo(DummyAction{
        log: "Hello GitHub!"
    })
}

func init() {
    // Register our DummyAction before main() gets run
    clicommon.RegisterAction(DummyAction{})
}

type DummyAction struct {
    log: string,
}

func (a DummyAction) Name() { return "dummyAction" }
func (a DummyAction) Params() { return []string{a.log} }

func (a DummyAction) Handle(params []string) error {
    fmt.Printf("Logging as superuser: %s", params[0])

    return nil
}
```

```
$ ./example
Logging as superuser: Hello Github!
```

### User config file helpers
```go
package main

import (
    fmt
    os

    clicommon "github.com/madwire-media/go-cli-common"
)

type Config struct {
    counter: int
}

func main() {
    // Use a user config folder named "my-app-name"
    configDir := clicommon.NewUserConfigDir("my-app-name")

    // Load 'myconfig.json' inside that folder and parse into the Config struct
    var config Config
    err := configDir.LoadConfig("myconfig", &config)
    if err != nil {
        fmt.Println("Error loading 'myconfig' file: %s", err)
        os.Exit(1)
    }

    config.counter++

    // Save the updated config back to 'myconfig.json' inside that folder
    err = configDir.SaveConfig("myconfig", &config)
    if err != nil {
        fmt.Println("Error saving 'myconfig' file: %s", err)
        os.Exit(1)
    }

    fmt.Printf("Incremented counter to %d", config.counter)
}
```

```
$ ./example
Incremented counter to 1
$ ./example
Incremented counter to 2
$ rm -r ~/.config/my-app-name
$ ./example
Incremented counter to 1
```

### CLI self-update system
```go
package main

import (
    fmt
    os

    clicommon "github.com/madwire-media/go-cli-common"
)

// This can be overridden by a linker, e.g. in a goreleaser build
var BuildVersion "0.0.1"
const GithubRepo "my-org/my-app-name"

func main() {
    maybeAutoUpdate()

    fmt.Println("Hello GitHub!")
    fmt.Println("Version:", BuildVersion)
}

func maybeAutoUpdate() {
    // Use the 'my-app-name' user config folder
    configDir := clicommon.NewUserConfigDir("my-app-name")

    // Set up a new auto-updater given our current build version, the repo to
    // update from, and that it's not private
    autoUpdater, err := clicommon.NewAutoUpdater(configDir, BuildVersion, GitHubRepo, false, nil)
    if err != nil {
        fmt.Println(err.Error())
    }

    // If there's an update available, download it, replace the current
    // executable, and re-execute it in place with the same arguments
    err = autoUpdater.TryAutoUpdateSelf()
    if err != nil {
        fmt.Println(err.Error())
    }
}
```

```
$ ./example
Automatic updating has not been configured, would you like to enable it? (only checks for updates every 24 hours)
Auto Update? (Y/n):
Checking for updates...
Hello Github!
Version: 0.0.1
$ ./example
Hello Github!
Version: 0.0.1
```
Then if you publish `v0.0.2` in GitHub Releases and wait 24 hours:
```
$ ./example
Checking for updates...
Updating to v0.0.2
Complete, restarting command...
Hello Github!
Version: 0.0.2
$ ./example
Hello Github!
Version: 0.0.2
```
