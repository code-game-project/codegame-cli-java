package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/Bananenpro/cli"
	"github.com/code-game-project/go-utils/cgfile"
	"github.com/code-game-project/go-utils/exec"
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

var artifactRegex = regexp.MustCompile(`^-\d+\.\d+\.\d+\.jar$`)

func buildClient(gameName, packageName, output, url string) (err error) {
	cli.BeginLoading("Building...")

	gameDir := filepath.Join("src", "main", "java")
	pkgDir := filepath.Join(strings.Split(packageName, ".")...)
	gameDir = filepath.Join(gameDir, pkgDir, toOneWord(gameName))

	err = replaceInFile(filepath.Join(gameDir, "Game.java"), "throw new RuntimeException(\"The CG_GAME_URL environment variable must be set.\")", "return \""+url+"\"")
	if err != nil {
		return err
	}
	defer func() {
		err2 := replaceInFile(filepath.Join(gameDir, "Game.java"), "return \""+url+"\"", "throw new RuntimeException(\"The CG_GAME_URL environment variable must be set.\")")
		if err == nil && err2 != nil {
			err = err2
		}
	}()

	err = os.RemoveAll("target")
	if err != nil {
		return fmt.Errorf("Failed to remove target directory: %w", err)
	}
	_, err = exec.Execute(true, "mvn", "-B", "clean", "package")
	if err != nil {
		return err
	}

	pkgParts := strings.Split(packageName, ".")
	artifactName := pkgParts[len(pkgParts)-1]

	if output != "" {
		outputIsFile := false
		if outStat, err := os.Stat(output); err != nil {
			if strings.HasSuffix(output, ".jar") {
				os.MkdirAll(filepath.Dir(output), 0o755)
				outputIsFile = true
			} else {
				os.MkdirAll(output, 0o755)
			}
		} else {
			outputIsFile = !outStat.IsDir()
		}

		files, err := os.ReadDir("target")
		if err != nil {
			return err
		}
		moved := false
		for _, f := range files {
			if f.IsDir() {
				continue
			}
			if strings.HasPrefix(f.Name(), artifactName) && artifactRegex.MatchString(strings.TrimPrefix(f.Name(), artifactName)) {
				if outputIsFile {
					err = os.Rename(filepath.Join("target", f.Name()), output)
					if err != nil {
						return fmt.Errorf("Couldn't move artifact to output location: %w", err)
					}
				} else {
					err = os.Rename(filepath.Join("target", f.Name()), filepath.Join(output, f.Name()))
					if err != nil {
						return fmt.Errorf("Couldn't move artifact to output location: %w", err)
					}
				}
				moved = true
				break
			}
		}
		if !moved {
			return errors.New("Couldn't move artifact to output location: failed to find generated artifact")
		}
	}

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
