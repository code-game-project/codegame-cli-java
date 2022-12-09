package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/code-game-project/go-utils/cgfile"
	cgExec "github.com/code-game-project/go-utils/exec"
	"github.com/code-game-project/go-utils/external"
	"github.com/code-game-project/go-utils/modules"
)

func Run() error {
	config, err := cgfile.LoadCodeGameFile("")
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

	data, err := modules.ReadCommandConfig[modules.RunData]()
	if err != nil {
		return err
	}

	url := external.TrimURL(config.URL)

	switch config.Type {
	case "client":
		return runClient(url, packageName, data.Args)
	default:
		return fmt.Errorf("Unknown project type: %s", config.Type)
	}
}

func runClient(url, packageName string, args []string) error {
	_, err := cgExec.Execute(true, "mvn", "compile")
	if err != nil {
		return err
	}

	for i, a := range args {
		a = strings.ReplaceAll(a, "'", "\\'")
		args[i] = "'" + a + "'"
	}

	cmdArgs := []string{"-e", "-q", "exec:java", "-Dexec.mainClass=" + packageName + ".App", "-Dexec.args=" + strings.Join(args, " ")}

	env := []string{"CG_GAME_URL=" + url}
	env = append(env, os.Environ()...)

	if _, err := exec.LookPath("mvn"); err != nil {
		return fmt.Errorf("`mvn` (Maven) ist not installed!")
	}

	cmd := exec.Command("mvn", cmdArgs...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = env

	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("Failed to run 'CG_GAME_URL=%s mvn %s'", url, strings.Join(cmdArgs, " "))
	}
	return nil
}
