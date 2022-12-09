package main

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/Bananenpro/cli"
	"github.com/code-game-project/go-utils/cgfile"
	"github.com/code-game-project/go-utils/modules"
)

func Build() error {
	config, err := cgfile.LoadCodeGameFile("")
	if err != nil {
		return err
	}

	data, err := modules.ReadCommandConfig[modules.BuildData]()
	if err != nil {
		return err
	}

	packageConf, ok := config.LangConfig["package"]
	if !ok {
		return errors.New("Missing language config field `package` in .codegame.json!")
	}
	packageName := packageConf.(string)
	if packageConf == "" {
		return errors.New("Empty language config field `package` in .codegame.json!")
	}

	if data.OS != "" || data.Arch != "" {
		return errors.New("Cross compilation is not supported for Java applications.")
	}

	switch config.Type {
	case "client":
		return buildClient(config.Game, packageName, data.Output, config.URL)
	default:
		return fmt.Errorf("Unknown project type: %s", config.Type)
	}
}

func buildClient(gameName, packageName, output, url string) error {
	cli.BeginLoading("Building...")
	// TODO
	cli.FinishLoading()
	return nil
}

func replaceInFile(filename, old, new string) error {
	content, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("Failed to replace '%s' with '%s' in '%s': %s", old, new, filename, err)
	}
	content = []byte(strings.ReplaceAll(string(content), old, new))
	err = os.WriteFile(filename, content, 0o644)
	if err != nil {
		return fmt.Errorf("Failed to replace '%s' with '%s' in '%s': %s", old, new, filename, err)
	}
	return nil
}
