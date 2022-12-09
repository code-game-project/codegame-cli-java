package main

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"

	_ "embed"

	"github.com/Bananenpro/cli"
	"github.com/code-game-project/go-utils/cgfile"
	"github.com/code-game-project/go-utils/cggenevents"
	"github.com/code-game-project/go-utils/exec"
	"github.com/code-game-project/go-utils/modules"
	"github.com/code-game-project/go-utils/server"
)

//go:embed templates/new/client/App.java.tmpl
var clientAppTemplate string

//go:embed templates/new/client/Game.java.tmpl
var clientGameTemplate string

//go:embed templates/new/client/pom.xml.tmpl
var clientPomXMLTemplate string

//go:embed templates/new/gitignore.tmpl
var gitignoreTemplate string

var packageRegexp = regexp.MustCompile(`[a-zA-Z](\.[a-zA-Z])*`)

func CreateNewClient(projectName string) error {
	data, err := modules.ReadCommandConfig[modules.NewClientData]()
	if err != nil {
		return err
	}

	api, err := server.NewAPI(data.URL)
	if err != nil {
		return err
	}

	packageName, err := cli.Input("Package name:", cli.Regexp(packageRegexp, "Invalid java package name. Valid example: com.example.myclient"))
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

	cgConf, err := cgfile.LoadCodeGameFile("")
	if err != nil {
		return err
	}
	cgConf.LangConfig["package"] = packageName
	err = cgConf.Write("")
	if err != nil {
		return err
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

	err = createClientTemplate(data.LibraryVersion, projectName, packageName, data.Name, info.DisplayName, eventNames, commandNames)
	if err != nil {
		return err
	}

	cli.BeginLoading("Installing java-client...")
	_, err = exec.Execute(true, "mvn", "-B", "dependency:resolve")
	if err != nil {
		return err
	}
	_, err = exec.Execute(true, "mvn", "-B", "dependency:resolve", "-Dclassifier=javadoc")
	if err != nil {
		return err
	}
	_, err = exec.Execute(true, "mvn", "-B", "dependency:sources")
	if err != nil {
		return err
	}
	cli.FinishLoading()
	return nil
}

func createClientTemplate(libraryVersion, projectName, packageName, gameName, displayName string, eventNames, commandNames []string) error {
	return execClientTemplate(libraryVersion, projectName, packageName, gameName, displayName, eventNames, commandNames, false)
}

func execClientTemplate(libraryVersion, projectName, packageName, gameName, displayName string, eventNames, commandNames []string, update bool) error {
	srcDir := filepath.Join(strings.Split(packageName, ".")...)
	srcDir = filepath.Join("src", "main", "java", srcDir)
	gameDir := filepath.Join(srcDir, toOneWord(gameName))
	if update {
		cli.Warn("This action will ERASE and regenerate ALL files in '%s/'.\nYou will have to manually update your code to work with the new version.", gameDir)
		ok, err := cli.YesNo("Continue?", false)
		if err != nil || !ok {
			return cli.ErrCanceled
		}
		os.RemoveAll(gameDir)
	} else {
		cli.Warn("DO NOT EDIT the `%s/` directory inside of the project. ALL CHANGES WILL BE LOST when running `codegame update`.", gameDir)
	}

	type event struct {
		Name       string
		PascalName string
	}

	events := make([]event, len(eventNames))
	for i, e := range eventNames {
		events[i] = event{
			Name:       e,
			PascalName: toPascal(e),
		}
	}

	commands := make([]event, len(commandNames))
	for i, c := range commandNames {
		commands[i] = event{
			Name:       c,
			PascalName: toPascal(c),
		}
	}

	groupID := "com.example"
	packageParts := strings.Split(packageName, ".")
	if len(packageParts) > 1 {
		groupID = strings.Join(packageParts[:len(packageParts)-1], ".")
	}

	libraryVersion += strings.Repeat(".0", 3-len(strings.Split(libraryVersion, ".")))

	data := struct {
		Package         string
		ProjectName     string
		GroupID         string
		ArtifactID      string
		DisplayName     string
		GameNameOneWord string
		Events          []event
		Commands        []event
		LibraryVersion  string
	}{
		Package:         packageName,
		GroupID:         groupID,
		ArtifactID:      packageParts[len(packageParts)-1],
		ProjectName:     projectName,
		DisplayName:     displayName,
		GameNameOneWord: toOneWord(gameName),
		Events:          events,
		Commands:        commands,
		LibraryVersion:  libraryVersion,
	}

	if !update {
		err := ExecTemplate(clientAppTemplate, filepath.Join(srcDir, "App.java"), data)
		if err != nil {
			return err
		}
		err = ExecTemplate(clientPomXMLTemplate, "pom.xml", data)
		if err != nil {
			return err
		}
		err = ExecTemplate(gitignoreTemplate, ".gitignore", data)
		if err != nil {
			return err
		}
	}

	err := ExecTemplate(clientGameTemplate, filepath.Join(gameDir, "Game.java"), data)
	if err != nil {
		return err
	}

	return nil
}

func toPascal(text string) string {
	text = strings.ReplaceAll(text, "_", " ")
	text = strings.ReplaceAll(text, "-", " ")
	text = strings.Title(text)
	text = strings.ReplaceAll(text, " ", "")
	return text
}

func toOneWord(text string) string {
	text = strings.ToLower(text)
	text = strings.ReplaceAll(text, "_", "")
	text = strings.ReplaceAll(text, "-", "")
	return text
}
