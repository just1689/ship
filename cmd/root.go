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

// ShipComponent contains the information necessary to install components into SHIP
type ShipComponent struct {
	Chart                helm.Chart
	PreInstallResources  []PreInstallResource
	PostInstallResources []PostInstallResource
}

// PreconditionReadySpec describes a resource to wait upon and the condition(s) of waiting
type PreconditionReadySpec struct {
	Resource    KubernetesResource
	MinReplicas int
}

// PreInstallResource contains the information necessary to execute a pre chart installation
// one-time manifest (e.g. configuration pod)
type PreInstallResource struct {
	PreconditionReady   PreconditionReadySpec
	ManifestPath        string
	WaitForDone         KubernetesResource
	PersistentAfterWait bool
}

// PostInstallResource contains the information necessary to execute a post chart installation
// one-time manifest (e.g. configuration pod)
type PostInstallResource struct {
	PreconditionReady   PreconditionReadySpec
	ManifestPath        string
	WaitForDone         KubernetesResource
	PersistentAfterWait bool
}

// KubernetesResource uniquely identifies a Kubernetes resource
type KubernetesResource struct {
	Name      string
	Type      string
	Namespace string
}

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

var shipComponents = []ShipComponent{
	{Chart: helm.Chart{ChartPath: "stable/postgresql", Namespace: "infra", ReleaseName: "inggwdb",
		ValuesPath: "resources/postgresql/values.yaml"}},
	{Chart: helm.Chart{ChartPath: "stable/prometheus-operator", Namespace: "infra", ReleaseName: "promop",
		ValuesPath: "resources/prometheus/values.yaml"}},
	{Chart: helm.Chart{ChartPath: "sprinthive-dev-charts/zipkin", Namespace: "infra", ReleaseName: "tracing",
		Overrides: []helm.ValueOverride{{Override: "ingress.enabled=true"}, {Override: "ingress.host=zipkin.${domain}"}, {Override: "ingress.class=kong"},
			{Override: "ingress.path=/"}}}},
	{Chart: helm.Chart{ChartPath: "stable/jenkins", Namespace: "infra", ReleaseName: "cicd",
    ValuesPath: "resources/jenkins/values.yaml",
		Overrides: []helm.ValueOverride{
			{Override: "Master.ingress.hostName=jenkins.${domain}"},
			{Override: "Master.ingress.tls[0].secretName=cicd-jenkins-tls"},
			{Override: "Master.ingress.tls[0].hosts[0]=jenkins.${domain}"}}}},
	{Chart: helm.Chart{ChartPath: "sprinthive-dev-charts/fluent-bit",
		Namespace: "infra", ReleaseName: "logcollect"}},
	{Chart: helm.Chart{ChartPath: "stable/grafana", Namespace: "infra", ReleaseName: "metricviz",
		Overrides: []helm.ValueOverride{
			{Override: "ingress.hosts={grafana.${domain}}"},
			{Override: "ingress.tls[0].secretName=grafana-tls"},
			{Override: "ingress.tls[0].hosts={grafana.${domain}}"}},
		ValuesPath: "resources/grafana/values.yaml"}},
	{Chart: helm.Chart{ChartPath: "sprinthive-dev-charts/kong", Namespace: "infra", ReleaseName: "inggw",
		Overrides: []helm.ValueOverride{{Override: "ProxyService.Type=ClusterIP"}, {Override: "IstioSidecar.enabled=false"},{Override: "HostPort=true"}}},
		PreInstallResources: []PreInstallResource{
			{
				PreconditionReady:   PreconditionReadySpec{Resource: KubernetesResource{Name: "kong-postgres", Type: "statefulset", Namespace: "infra"}, MinReplicas: 1},
				ManifestPath:        "resources/kong/pod-kong-pre-configure.yaml",
				WaitForDone:         KubernetesResource{Name: "kong-pre-configure", Type: "pod", Namespace: "infra"},
				PersistentAfterWait: false}},
		PostInstallResources: []PostInstallResource{
			{
				PreconditionReady:   PreconditionReadySpec{Resource: KubernetesResource{Name: "inggw-kong", Type: "daemonset", Namespace: "infra"}, MinReplicas: 1},
				ManifestPath:        "resources/kong/pod-kong-configure.yaml",
				WaitForDone:         KubernetesResource{Name: "kong-configure", Type: "pod", Namespace: "infra"},
				PersistentAfterWait: false}}},
	{Chart: helm.Chart{ChartPath: "sprinthive-dev-charts/kong-ingress-controller", Namespace: "infra", ReleaseName: "ingcontrol"}},
	{Chart: helm.Chart{ChartPath: "stable/cert-manager", Namespace: "cert-manager", ReleaseName: "certman", Overrides: []helm.ValueOverride{{Override: "ingressShim.defaultIssuerName=letsencrypt-prod"}, {Override: "ingressShim.defaultIssuerKind=ClusterIssuer"}}},
		PostInstallResources: []PostInstallResource{{ManifestPath: "resources/cert-manager/clusterissuer-letsencrypt-prod.yaml", PersistentAfterWait: true}}},
}

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
	viper.SetDefault("components", shipComponents)
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
