package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gonuts/commander"
	"github.com/gonuts/flag"
	"io"
	"io/ioutil"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

func makeRegisterCmd() *commander.Command {
	var registerCmd = &commander.Command{
		Run:       runRegisterCmd,
		UsageLine: "register [options]",
		Flag:      *flag.NewFlagSet("mezasi-register", flag.ExitOnError),
	}

	registerCmd.Flag.String("name", "", "vm name (* required)")
	registerCmd.Flag.String("base", "", "base image (*)")
	registerCmd.Flag.String("public-key", "", "path to ssh public key file (for root@vm)")
	registerCmd.Flag.String("user-data", "", "path to shell script executed on boot up")
	registerCmd.Flag.Bool("wait", false, "wait for vm boot up")
	return registerCmd
}

var listCmd = &commander.Command{
	Run:       runListCmd,
	UsageLine: "list",
}

var configCmd = &commander.Command{
	Run:       runConfigCmd,
	UsageLine: "config",
}

var infoCmd = &commander.Command{
	Run:       runInfoCmd,
	UsageLine: "info <name>",
}

var startCmd = &commander.Command{
	Run:       runPostHandleNameCmd,
	UsageLine: "start <name>",
}

var stopCmd = &commander.Command{
	Run:       runPostHandleNameCmd,
	UsageLine: "stop <name>",
}

var forceStopCmd = &commander.Command{
	Run:       runPostHandleNameCmd,
	UsageLine: "force_stop <name>",
}

func makeRemoveCmd() *commander.Command {
	var removeCmd = &commander.Command{
		Run:       runRemoveCmd,
		UsageLine: "remove <name>",
		Flag:      *flag.NewFlagSet("mezasi-remove", flag.ExitOnError),
	}
	removeCmd.Flag.Bool("yes", false, "")
	return removeCmd
}

var publicKeyCmd = &commander.Command{
	Run:       runPostFileHandleNameCmd,
	UsageLine: "public_key <name> [file]",
}

var userDataCmd = &commander.Command{
	Run:       runPostFileHandleNameCmd,
	UsageLine: "user_data <name> [file]",
}

var sshCmd = &commander.Command{
	Run:       runSshCmd,
	UsageLine: "ssh <name> ...",
}

func runListCmd(cmd *commander.Command, args []string) error {
	if err := validateCmdArgs(cmd, args); err != nil {
		return err
	}
	req, err := client.NewRequest("GET", "vm/list", nil)
	if err != nil {
		return err
	}
	if _, err = pp(client.Do(req)); err != nil {
		return err
	}
	return nil
}

func runConfigCmd(cmd *commander.Command, args []string) error {
	if err := validateCmdArgs(cmd, args); err != nil {
		return err
	}
	req, err := client.NewRequest("GET", "config", nil)
	if err != nil {
		return err
	}
	if _, err := pp(client.Do(req)); err != nil {
		return err
	}
	return nil
}

func runInfoCmd(cmd *commander.Command, args []string) error {
	if err := validateCmdArgs(cmd, args); err != nil {
		return err
	}
	req, err := client.NewRequest("GET", "vm/info/"+args[0], nil)
	if err != nil {
		return err
	}
	if _, err := pp(client.Do(req)); err != nil {
		return err
	}
	return nil
}

func runPostHandleNameCmd(cmd *commander.Command, args []string) error {
	if err := validateCmdArgs(cmd, args); err != nil {
		return err
	}
	req, err := client.NewRequest("POST", "vm/"+cmd.Name()+"/"+args[0], nil)
	if err != nil {
		return err
	}
	if _, err := pp(client.Do(req)); err != nil {
		return err
	}
	return nil
}

func runRemoveCmd(cmd *commander.Command, args []string) error {
	if err := validateCmdArgs(cmd, args); err != nil {
		return err
	}
	yes := cmd.Flag.Lookup("yes").Value.Get().(bool)
	name := args[0]

	if !yes {
		fmt.Printf("Really remove %s? [y/N]\n", name)
		var y string
		fmt.Scanf("%s", &y)
		if y != "y" && y != "Y" {
			os.Exit(1)
		}
	}
	req, err := client.NewRequest("POST", "vm/remove/"+name, nil)
	if err != nil {
		return err
	}
	if _, err := pp(client.Do(req)); err != nil {
		return err
	}
	return nil
}

func runRegisterCmd(cmd *commander.Command, args []string) error {
	if err := validateCmdArgs(cmd, args); err != nil {
		return err
	}
	name := cmd.Flag.Lookup("name").Value.Get().(string)
	base := cmd.Flag.Lookup("base").Value.Get().(string)
	if name == "" || base == "" {
		return errors.New("Usage: " + cmd.UsageLine)
	}

	var b bytes.Buffer
	w := multipart.NewWriter(&b)
	if err := w.WriteField("name", name); err != nil {
		return err
	}
	if err := w.WriteField("base", base); err != nil {
		return err
	}

	if publicKey := cmd.Flag.Lookup("public-key").Value.Get().(string); publicKey != "" {
		if err := writeFileField(w, "public_key", publicKey); err != nil {
			return err
		}
	}
	if userData := cmd.Flag.Lookup("user-data").Value.Get().(string); userData != "" {
		if err := writeFileField(w, "user_data", userData); err != nil {
			return err
		}
	}
	if err := w.Close(); err != nil {
		return err
	}

	req, err := client.NewRequest("POST", "vm/register", &b)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", w.FormDataContentType())

	if _, err := pp(client.Do(req)); err != nil {
		return err
	}

	wait := cmd.Flag.Lookup("wait").Value.Get().(bool)
	if !wait {
		return nil
	}
	log.Println("waiting for vm boot up...")

	req, err = client.NewRequest("GET", "notify/"+name, nil)
	if err != nil {
		return err
	}
	if _, err = pp(client.Do(req)); err != nil {
		return err
	}
	return nil
}

func runPostFileHandleNameCmd(cmd *commander.Command, args []string) error {
	if err := validateCmdArgs(cmd, args); err != nil {
		return err
	}
	cmdName := cmd.Name()
	pathStr := cmdName + "/" + args[0]
	switch len(args) {
	case 1:
		req, err := client.NewRequest("GET", pathStr, nil)
		if err != nil {
			return err
		}
		if _, err := pp(client.Do(req)); err != nil {
			return err
		}
	case 2:
		var b bytes.Buffer
		w := multipart.NewWriter(&b)
		if err := writeFileField(w, cmdName, args[1]); err != nil {
			return err
		}
		if err := w.Close(); err != nil {
			return err
		}

		req, err := client.NewRequest("POST", pathStr, &b)
		if err != nil {
			return err
		}
		req.Header.Set("Content-Type", w.FormDataContentType())

		if _, err := pp(client.Do(req)); err != nil {
			return err
		}
	default:
		return errors.New("invalid arguments")
	}
	return nil
}

func runSshCmd(cmd *commander.Command, args []string) error {
	if err := validateCmdArgs(cmd, args); err != nil {
		return err
	}
	req, err := client.NewRequest("GET", "vm/info/"+args[0], nil)
	if err != nil {
		return err
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode != 200 {
		pp(resp, err)
		return err
	}

	dec := json.NewDecoder(resp.Body)
	defer resp.Body.Close()

	m := make(map[string]interface{})
	if err := dec.Decode(&m); err != nil {
		return err
	}
	cmdPath, err := exec.LookPath("ssh")
	if err != nil {
		return err
	}
	execArgs := []string{m["ip_addr"].(string)}
	execArgs = append(execArgs, args[1:]...)

	execCmd := exec.Command(cmdPath, execArgs...)
	execCmd.Stdin = os.Stdin
	execCmd.Stdout = os.Stdout
	execCmd.Stderr = os.Stderr

	return execCmd.Run()
}

func validateCmdArgs(cmd *commander.Command, args []string) error {
	mustVal := 0
	optionVal := 0
	must := regexp.MustCompile(`<\w+>`)
	option := regexp.MustCompile(`\[\w+\]`)

	for _, action := range strings.Split(cmd.UsageLine, " ") {
		if must.MatchString(action) {
			mustVal++
		}
		if option.MatchString(action) {
			optionVal++
		}
		// skip arguments validation
		if action == "..." {
			if mustVal <= len(args) {
				return nil
			}
		}
	}
	if mustVal <= len(args) && len(args) <= mustVal+optionVal {
		return nil
	}
	return errors.New("invalid argument\nUsage: " + cmd.UsageLine)
}

func writeFileField(w *multipart.Writer, fieldName string, filePath string) error {
	ff, err := w.CreateFormField(fieldName)
	if err != nil {
		return err
	}
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	if _, err := io.Copy(ff, file); err != nil {
		return err
	}
	return nil
}

func pp(resp *http.Response, err error) (*http.Response, error) {
	if err != nil {
		return resp, err
	}
	log.Println(resp.Status)

	defer resp.Body.Close()
	src, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return resp, err
	}

	index := strings.Index(resp.Header.Get("Content-Type"), "json")

	if index > 0 {
		var b bytes.Buffer
		err := json.Indent(&b, src, "", "    ")
		if err != nil {
			return resp, err
		}
		fmt.Println(b.String())
	} else {
		fmt.Println(string(src))
	}
	return resp, err
}
