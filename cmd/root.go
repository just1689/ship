// Copyright © 2017 NAME HERE <EMAIL ADDRESS>
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/SprintHive/ship/pkg/helm"
	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:   "ship",
	Short: "A brief description of your application",
	Long: `A longer description that spans multiple lines and likely contains
examples and usage of using your application. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	//	Run: func(cmd *cobra.Command, args []string) { },
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := RootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

var defaultCharts = []helm.Chart{
	helm.Chart{ChartPath: "sprinthive-dev-charts/kong-cassandra", Namespace: "infra", ReleaseName: "inggwdb",
		Overrides: []string{"clusterProfile=local"}, ValuesPath: ""},
	helm.Chart{ChartPath: "sprinthive-dev-charts/nexus", Namespace: "infra", ReleaseName: "repo", Overrides: []string{},
		ValuesPath: ""},
	helm.Chart{ChartPath: "sprinthive-dev-charts/prometheus", Namespace: "infra", ReleaseName: "metricdb",
		Overrides: []string{}, ValuesPath: ""},
	helm.Chart{ChartPath: "sprinthive-dev-charts/zipkin", Namespace: "infra", ReleaseName: "tracing",
		Overrides: []string{"ingress.enabled=true", "ingress.host=zipkin.${domain}", "ingress.class=kong", "ingress.path=/"}, ValuesPath: "",
	},
	helm.Chart{ChartPath: "sprinthive-dev-charts/jenkins", Namespace: "infra", ReleaseName: "cicd", Overrides: []string{"Master.HostName=jenkins.${domain}"}, ValuesPath: ""},
	helm.Chart{ChartPath: "sprinthive-dev-charts/kibana", Namespace: "infra", ReleaseName: "logviz",
		Overrides:  []string{"ingress.enabled=true", "ingress.host=kibana.${domain}", "ingress.class=kong", "ingress.path=/"},
		ValuesPath: ""},
	helm.Chart{ChartPath: "sprinthive-dev-charts/fluent-bit",
		Namespace: "infra", ReleaseName: "logcollect", Overrides: []string{}, ValuesPath: ""},
	helm.Chart{ChartPath: "sprinthive-dev-charts/elasticsearch", Namespace: "infra", ReleaseName: "logdb", Overrides: []string{"ClusterProfile=local"}, ValuesPath: ""},
	helm.Chart{ChartPath: "stable/grafana", Namespace: "infra", ReleaseName: "metricviz", Overrides: []string{"server.ingress.enabled=true", "server.ingress.hosts={grafana.${domain}}"}, ValuesPath: "resources/grafana/values.yaml"},
	helm.Chart{ChartPath: "sprinthive-dev-charts/kong", Namespace: "infra", ReleaseName: "inggw",
		Overrides: []string{"clusterProfile=local", "HostPort=true"}, ValuesPath: ""},
	helm.Chart{ChartPath: "sprinthive-dev-charts/kong-ingress-controller", Namespace: "infra", ReleaseName: "ingcontrol", Overrides: []string{}, ValuesPath: ""}}

func init() {
	cobra.OnInitialize(initConfig)

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.
	RootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.ship.yaml)")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	RootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")

	checkDependencies()
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	setDefaults()

	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := homedir.Dir()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		// Search config in home directory with name ".ship" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigName(".ship")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}

func setDefaults() {
	viper.SetDefault("charts", defaultCharts)
}

func checkDependencies() {
	cmdName := "helm"
	args := []string{"version"}

	if output, err := exec.Command(cmdName, args...).CombinedOutput(); err != nil {
		if len(output) == 0 {
			fmt.Fprintf(os.Stderr, "Helm is not installed. Please see https://github.com/kubernetes/helm for instructions on how to install helm.\n")
			os.Exit(1)
		}
		outputMsg := string(output)
		if strings.Contains(outputMsg, "Error") {
			fmt.Fprintf(os.Stderr, fmt.Sprintf(`Helm installation is not healthy:

%s
Please see https://github.com/kubernetes/helm for instructions on how to install helm correctly.`,
				outputMsg))
			os.Exit(1)
		}
	}
}
