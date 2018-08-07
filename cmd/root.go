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
	{Chart: helm.Chart{ChartPath: "stable/heapster", Namespace: "kube-system", ReleaseName: "sysmetric",
  Overrides: []helm.ValueOverride{{Override: "rbac.create=true"}}}},
	{Chart: helm.Chart{ChartPath: "sprinthive-dev-charts/postgresql", Namespace: "infra", ReleaseName: "inggwdb",
		Overrides: []helm.ValueOverride{{Override: "fullnameOverride=kong-postgres"},{Override: "postgresUser=kong"},{Override: "postgresDatabase=kong"}}}},
	{Chart: helm.Chart{ChartPath: "sprinthive-dev-charts/nexus", Namespace: "infra", ReleaseName: "repo"}},
	{Chart: helm.Chart{ChartPath: "stable/prometheus", Namespace: "infra", ReleaseName: "metricdb",
		ValuesPath: "resources/prometheus/values.yaml"}},
	{Chart: helm.Chart{ChartPath: "sprinthive-dev-charts/zipkin", Namespace: "infra", ReleaseName: "tracing",
		Overrides: []helm.ValueOverride{{Override: "ingress.enabled=true"}, {Override: "ingress.host=zipkin.${domain}"}, {Override: "ingress.class=kong"},
			{Override: "ingress.path=/"}}}},
	{Chart: helm.Chart{ChartPath: "sprinthive-dev-charts/jenkins", Namespace: "infra", ReleaseName: "cicd",
		Overrides: []helm.ValueOverride{
			{Override: "Master.HostName=jenkins.${domain}"},
			{Override: "Master.Ingress.Annotations.kubernetes\\.io/ingress\\.class=kong"},
			{Override: "Master.Ingress.TLS[0].secretName=cicd-jenkins-tls"},
			{Override: "Master.Ingress.TLS[0].hosts[0]=jenkins.${domain}"}}}},
	{Chart: helm.Chart{ChartPath: "sprinthive-dev-charts/kibana", Namespace: "infra", ReleaseName: "logviz",
		Overrides: []helm.ValueOverride{
			{Override: "ingress.enabled=true"},
			{Override: "ingress.host=kibana.${domain}"},
			{Override: "ingress.class=kong"},
			{Override: "ingress.path=/"}}}},
	{Chart: helm.Chart{ChartPath: "sprinthive-dev-charts/fluent-bit",
		Namespace: "infra", ReleaseName: "logcollect"}},
	{Chart: helm.Chart{ChartPath: "sprinthive-dev-charts/elasticsearch", Namespace: "infra", ReleaseName: "logdb", Overrides: []helm.ValueOverride{
		{Override: "ClusterProfile=production"}}}},
	{Chart: helm.Chart{ChartPath: "stable/grafana", Namespace: "infra", ReleaseName: "metricviz",
		Overrides: []helm.ValueOverride{
			{Override: "sidecar.dashboards.enabled=true"},
			{Override: "sidecar.dashboards.label=grafana_dashboard"},
			{Override: "sidecar.datasources.enabled=true"},
			{Override: "sidecar.datasources.label=grafana_datasource"},
			{Override: "ingress.enabled=true"},
			{Override: "ingress.hosts={grafana.${domain}}"},
			{Override: "ingress.tls[0].hosts={grafana.${domain}}"}},
		ValuesPath: "resources/grafana/values.yaml"},
		PreInstallResources: []PreInstallResource{
			{ManifestPath: "resources/grafana/datasource-elasticsearch.yaml", PersistentAfterWait: true},
			{ManifestPath: "resources/grafana/datasource-prometheus.yaml", PersistentAfterWait: true},
			{ManifestPath: "resources/grafana/dashboard-cluster-alarms.yaml", PersistentAfterWait: true},
			{ManifestPath: "resources/grafana/dashboard-ingress.yaml", PersistentAfterWait: true},
			{ManifestPath: "resources/grafana/dashboard-kubernetes.yaml", PersistentAfterWait: true},
			{ManifestPath: "resources/grafana/dashboard-pod-alarms.yaml", PersistentAfterWait: true}}},
	{Chart: helm.Chart{ChartPath: "sprinthive-dev-charts/kong", Namespace: "infra", ReleaseName: "inggw",
		Overrides: []helm.ValueOverride{{Override: "ProxyService.Type=ClusterIP"}, {Override: "IstioSidecar.enabled=false"},{Override: "HostPort=true"}}},
		PreInstallResources: []PreInstallResource{
			{
				PreconditionReady:   PreconditionReadySpec{Resource: KubernetesResource{Name: "kong-postgres", Type: "deployment", Namespace: "infra"}, MinReplicas: 1},
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
	{Chart: helm.Chart{ChartPath: "stable/cert-manager", Namespace: "infra", ReleaseName: "certman", Overrides: []helm.ValueOverride{{Override: "ingressShim.defaultIssuerName=letsencrypt-prod"}, {Override: "ingressShim.defaultIssuerKind=ClusterIssuer"}}},
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
