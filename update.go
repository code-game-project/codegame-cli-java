package main

import (
	"errors"
	"fmt"

	"github.com/Bananenpro/cli"
	"github.com/code-game-project/go-utils/cgfile"
	"github.com/code-game-project/go-utils/cggenevents"
	"github.com/code-game-project/go-utils/exec"
	"github.com/code-game-project/go-utils/modules"
	"github.com/code-game-project/go-utils/server"
)

func Update(projectName string) error {
	config, err := cgfile.LoadCodeGameFile("")
	if err != nil {
		return err
	}

	data, err := modules.ReadCommandConfig[modules.UpdateData]()
	if err != nil {
		return err
	}
	switch config.Type {
	case "client":
		return updateClient(projectName, data.LibraryVersion, config)
	default:
		return fmt.Errorf("Unknown project type: %s", config.Type)
	}
}

func updateClient(projectName, libraryVersion string, config *cgfile.CodeGameFileData) error {
	api, err := server.NewAPI(config.URL)
	if err != nil {
		return err
	}

	info, err := api.FetchGameInfo()
	if err != nil {
		return err
	}
	if info.DisplayName == "" {
		info.DisplayName = info.Name
	}

	cge, err := api.GetCGEFile()
	if err != nil {
		return err
	}
	cgeVersion, err := cggenevents.ParseCGEVersion(cge)
	if err != nil {
		return err
	}

	eventNames, commandNames, err := cggenevents.GetEventNames(api.BaseURL(), cgeVersion)
	if err != nil {
		return err
	}

	cgConf, err := cgfile.LoadCodeGameFile("")
	if err != nil {
		return err
	}
	packageConf, ok := cgConf.LangConfig["package"]
	if !ok {
		return errors.New("Missing language config field `package` in .codegame.json!")
	}
	packageName := packageConf.(string)
	if packageConf == "" {
		return errors.New("Empty language config field `package` in .codegame.json!")
	}

	err = updateClientTemplate(libraryVersion, projectName, packageName, config.Game, info.DisplayName, eventNames, commandNames)
	if err != nil {
		return err
	}

	cli.BeginLoading("Updating java-client...")
	exec.Execute(true, "mvn", "dependency:resolve")
	exec.Execute(true, "mvn", "dependency:sources")
	exec.Execute(true, "mvn", "dependency:resolve", "-Dclassifier=javadoc")
	cli.FinishLoading()
	return nil
}

func updateClientTemplate(libraryVersion, projectName, packageName, gameName, displayName string, eventNames, commandNames []string) error {
	return execClientTemplate(libraryVersion, projectName, packageName, gameName, displayName, eventNames, commandNames, true)
}
