package helm

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

var (
	cmdName = "helm"
)

// Chart contains the information needed to install a helm chart
type Chart struct {
	ChartPath   string
	Namespace   string
	ReleaseName string
	Overrides   []string
	ValuesPath  string
}

// GetHelmReleases returns the list of helm releases in the kubernetes cluster in the active profile
func GetHelmReleases() []string {
	args := []string{"list", "-q"}

	output, err := exec.Command(cmdName, args...).CombinedOutput()
	if err != nil {
		panic(fmt.Sprintf("Failed to remove charts: %v", string(output)))
	}

	releases := strings.Split(strings.Trim(string(output), "\" "), "\n")
	// The last line is always empty, so pop it
	releases = releases[:len(releases)-1]

	return releases
}

// RemoveReleases removes the releases created by the given charts
func RemoveReleases(sourceCharts *[]Chart) {
	currentReleases := GetHelmReleases()
	currentReleasesMap := make(map[string]struct{})
	for _, currentRelease := range currentReleases {
		currentReleasesMap[currentRelease] = struct{}{}
	}

	for _, sourceChart := range *sourceCharts {
		if _, found := currentReleasesMap[sourceChart.ReleaseName]; found {
			fmt.Println(fmt.Sprintf("Removing release: %v", sourceChart.ReleaseName))
			removeHelmRelease(sourceChart.ReleaseName)
		}
	}
}

// InstallChartRepo installs a given helm repository into the repository config
func InstallChartRepo() {
	args := []string{"repo", "add", "sprinthive-dev-charts", "https://s3.eu-west-2.amazonaws.com/sprinthive-dev-charts"}

	if output, err := exec.Command(cmdName, args...).CombinedOutput(); err != nil {
		fmt.Fprintf(os.Stderr, fmt.Sprintf("Failed to install sprinthive charts: %v", string(output)))
		os.Exit(1)
	}

	fmt.Println("Successfully installed sprinthive chart repo")
}

// InstallCharts will install the provided charts into the currently configured Kubernetes cluster
func InstallCharts(charts *[]Chart, domain string) {
	for _, chart := range *charts {
		fmt.Printf("installing chart: %s\n", chart.ChartPath)
		args := []string{"install", chart.ChartPath, "-n", chart.ReleaseName, "--namespace", chart.Namespace}

		for _, override := range chart.Overrides {
			args = append(args, "--set", strings.Replace(override, "${domain}", domain, -1))
		}

		if chart.ValuesPath != "" {
			args = append(args, "--values", chart.ValuesPath)
		}

		if output, err := exec.Command(cmdName, args...).CombinedOutput(); err != nil {
			panic(fmt.Sprintf("Failed to install chart: %v", string(output)))
		}
	}
}

func removeHelmRelease(releaseName string) {
	args := []string{"delete", "--purge", releaseName}

	if output, err := exec.Command(cmdName, args...).CombinedOutput(); err != nil {
		panic(fmt.Sprintf("Failed to remove helm release '%s': %v", releaseName, string(output)))
	}
}
