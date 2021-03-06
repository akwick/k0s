package cmd

import (
	"fmt"
	"os"
	"reflect"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/k0sproject/k0s/pkg/apis/v1beta1"
	"github.com/k0sproject/k0s/pkg/install"
)

func init() {
	installCmd.Flags().StringVar(&role, "role", "server", "node role (possible values: server or worker. In a single-node setup, a worker role should be used)")
}

var (
	role string

	installCmd = &cobra.Command{
		Use:   "install",
		Short: "Helper command for setting up k0s on a brand-new system. Must be run as root (or with sudo)",
		RunE: func(cmd *cobra.Command, args []string) error {
			switch role {
			case "server", "worker":
				return setup()
			default:
				logrus.Errorf("invalid value %s for install role", role)
				return cmd.Help()
			}
		},
	}
)

// the setup functions:
// * Ensures that the proper users are created
// * sets up startup and logging for k0s
func setup() error {
	if os.Geteuid() != 0 {
		logrus.Fatal("this command must be run as root!")
	}

	if role == "server" {
		if err := createServerUsers(); err != nil {
			logrus.Errorf("failed to create server users: %v", err)
		}
	}
	// set-up service and logging
	serviceArgs := getCmdArgs(role)
	err := install.EnsureService(serviceArgs)
	if err != nil {
		logrus.Errorf("failed to install k0s service: %v", err)
	}
	return nil
}

func createServerUsers() error {
	clusterConfig, err := ConfigFromYaml(cfgFile)
	if err != nil {
		return err
	}
	users := getUserList(*clusterConfig.Install.SystemUsers)

	var messages []string
	for _, v := range users {
		if err := install.EnsureUser(v, k0sVars.DataDir); err != nil {
			messages = append(messages, err.Error())
		}
	}

	if len(messages) > 0 {
		return fmt.Errorf(strings.Join(messages, "\n"))
	}
	return nil
}

func getUserList(sysUsers v1beta1.SystemUser) []string {
	v := reflect.ValueOf(sysUsers)
	values := make([]string, v.NumField())

	for i := 0; i < v.NumField(); i++ {
		values[i] = v.Field(i).String()
	}
	return values
}

func getCmdArgs(role string) []string {
	var args []string

	if role == "server" {
		args = append(args, "server")
		if cfgFile != "" {
			args = append(args, "--config", cfgFile)
		}
	} else {
		args = append(args, "worker", "--token-file", "REPLACEME")
	}

	if debug {
		args = append(args, "--debug")
	}
	return args
}
