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
	"strings"
	"time"

	"github.com/spf13/cobra"
)

// HelmChart contains the information needed to install a helm chart
type HelmChart struct {
	ChartPath   string
	Namespace   string
	ReleaseName string
	Overrides   []string
}

var defaultCharts = []HelmChart{
	HelmChart{"sprinthive-dev-charts/kong-cassandra", "infra", "inggwdb", []string{"clusterProfile=local"}},
	HelmChart{"sprinthive-dev-charts/nexus", "infra", "repo", []string{}},
	HelmChart{"sprinthive-dev-charts/prometheus", "infra", "metricdb", []string{}},
	HelmChart{"sprinthive-dev-charts/zipkin", "infra", "tracing", []string{}},
	HelmChart{"sprinthive-dev-charts/jenkins", "infra", "cicd", []string{}},
	HelmChart{"sprinthive-dev-charts/kibana", "infra", "logviz", []string{}},
	HelmChart{"sprinthive-dev-charts/fluent-bit", "infra", "logcollect", []string{}},
	HelmChart{"sprinthive-dev-charts/elasticsearch", "infra", "logdb", []string{"ClusterProfile=local"}},
	HelmChart{"stable/grafana", "infra", "metricviz", []string{}},
	HelmChart{"sprinthive-dev-charts/kong", "infra", "inggw",
		[]string{"clusterProfile=local", "ProxyService.Type=NodePort"}}}

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
		checkDependencies()
		installChartRepo()
		installCharts(&defaultCharts)
		configureGrafana()
	},
}

func configureGrafana() {
	waitDeployReady("metricviz-grafana", 1, "infra")

	cmd := "kubectl"
	args := []string{"create", "-f", "resources/pod-grafana-configure.yaml"}

	if output, err := exec.Command(cmd, args...).CombinedOutput(); err != nil {
		fmt.Fprintf(os.Stderr, fmt.Sprintf("Failed to configure grafana: %v", string(output)))
	}
}

func checkDependencies() {
	cmd := "helm"
	args := []string{"version"}

	if output, err := exec.Command(cmd, args...).CombinedOutput(); err != nil {
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

func installChartRepo() {
	fmt.Println("install chart repo called")

	cmd := "helm"
	args := []string{"repo", "add", "sprinthive-dev-charts", "https://s3.eu-west-2.amazonaws.com/sprinthive-dev-charts"}

	if output, err := exec.Command(cmd, args...).CombinedOutput(); err != nil {
		fmt.Fprintf(os.Stderr, fmt.Sprintf("Failed to install sprinthive charts: %v", string(output)))
		os.Exit(1)
	}

	fmt.Println("Successfully installed sprinthive chart repo")
}

func installCharts(charts *[]HelmChart) {
	cmd := "helm"

	for _, chart := range *charts {
		fmt.Printf("installing chart: %s\n", chart.ChartPath)
		args := []string{"install", chart.ChartPath, "-n", chart.ReleaseName, "--namespace", chart.Namespace}

		for _, override := range chart.Overrides {
			args = append(args, "--set", override)
		}

		if output, err := exec.Command(cmd, args...).CombinedOutput(); err != nil {
			panic(fmt.Sprintf("Failed to install chart: %v", string(output)))
		}
	}
}

func waitDeployReady(deployName string, expectedNumberReady int, namespace string) {
	cmd := "kubectl"

	fmt.Printf("Waiting for readiness of deploy %s\n", deployName)
	args := []string{"get", "deploy", "-o", "jsonpath=\"{range .items[*]}{@.metadata.name}:{@.status.readyReplicas}{end}\"", "--namespace", namespace}

	deployFinished := false
	for !deployFinished {
		time.Sleep(1000 * time.Millisecond)

		output, err := exec.Command(cmd, args...).CombinedOutput()
		if err != nil {
			panic(fmt.Sprintf("Failed execute kubectl command to get deploy status: %v", string(output)))
		}

		if strings.Contains(string(output), fmt.Sprintf("%s:%d", deployName, expectedNumberReady)) {
			fmt.Println("Deploy is ready!")
			deployFinished = true
		}
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
}
