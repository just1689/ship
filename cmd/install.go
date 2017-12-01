// Copyright © 2017 SprintHive (Pty) Ltd (buzz@sprinthive.com)
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

	"github.com/SprintHive/ship/pkg/helm"
	"github.com/SprintHive/ship/pkg/kubectl"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// installCmd represents the create command
var installCmd = &cobra.Command{
	Use:   "install",
	Short: "Installs the SHIP components into your Kubernetes cluster",
	Long: `Install a bundle of SHIP components into your Kubernetes cluster using helm.
	
	The following components will be installed:
	* Ingress GW (Kong)
	* Ingress GW Database (Cassandra)
	* Logging database (Elasticsearch)
	* Log collector (Fluent-bit)
	* Tracing (Zipkin)
	* Metric database (Prometheus)
	* Metric Visualization (Grafana)
	* Log Viewer (Kibana)
	* CI/CD (Jenkins)
	* Artifact repository (Nexus)`,
	Run: func(cmd *cobra.Command, args []string) {
		domain, err := cmd.Flags().GetString("domain")
		if err != nil {
			fmt.Fprintf(os.Stderr, fmt.Sprintf("Failed to get domain flag"))
			os.Exit(1)
		}
		var charts []helm.Chart
		viper.UnmarshalKey("charts", &charts)

		helm.InstallChartRepo()
		helm.InstallCharts(&charts, domain)
		configureGrafana()
		configureKong()
	},
}

func configureGrafana() {
	kubectl.WaitDeployReady("metricviz-grafana", 1, "infra")
	kubectl.Create("resources/grafana/cm-grafana-datasources.yaml", "infra")
	kubectl.Create("resources/grafana/cm-grafana-dashboards.yaml", "infra")
	kubectl.Create("resources/grafana/pod-grafana-configure.yaml", "infra")
	kubectl.WaitPodCompleted("grafana-configure", "infra")

	// Clean up pod
	cmdName := "kubectl"
	args := []string{"delete", "pod", "grafana-configure", "--namespace", "infra"}

	if output, err := exec.Command(cmdName, args...).CombinedOutput(); err != nil {
		fmt.Fprintf(os.Stderr, fmt.Sprintf("Failed to delete grafana-configure pod: %v", string(output)))
	}

	// Clean up config maps
	args = []string{"delete", "configmap", "grafana-dashboards", "grafana-datasources", "--namespace", "infra"}

	if output, err := exec.Command(cmdName, args...).CombinedOutput(); err != nil {
		fmt.Fprintf(os.Stderr, fmt.Sprintf("Failed to delete config maps for grafana-configure: %v", string(output)))
	}
}

func configureKong() {
	kubectl.WaitDaemonSetReady("inggw-kong", 1, "infra")
	kubectl.Create("resources/kong/pod-kong-configure.yaml", "infra")
	kubectl.WaitPodCompleted("kong-configure", "infra")

	// Clean up pod
	cmdName := "kubectl"
	args := []string{"delete", "pod", "kong-configure", "--namespace", "infra"}

	if output, err := exec.Command(cmdName, args...).CombinedOutput(); err != nil {
		fmt.Fprintf(os.Stderr, fmt.Sprintf("Failed to delete kong-configure pod: %v", string(output)))
	}
}

func init() {
	RootCmd.AddCommand(installCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// installCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// installCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
	installCmd.Flags().StringP("domain", "d", "", "Sets the base domain that will be used for ingress. *.<base domain> should resolve to your Kubernetes cluster.")
	installCmd.MarkFlagRequired("domain")
}
