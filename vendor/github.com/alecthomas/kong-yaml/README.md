# A Kong configuration loader for YAML [![](https://godoc.org/github.com/alecthomas/kong-yaml?status.svg)](http://godoc.org/github.com/alecthomas/kong-yaml) [![CircleCI](https://img.shields.io/circleci/project/github/alecthomas/kong-yaml.svg)](https://circleci.com/gh/alecthomas/kong-yaml)

Use it like so:

```go
parser, err := kong.New(&cli, kong.Configuration(kongyaml.Loader, "/etc/myapp/config.yaml", "~/.myapp.yaml))
```