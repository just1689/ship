package kubectl

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

var (
	cmdName = "kubectl"
)

// Create adds the given resource to the active kubernetes cluster
func Create(filePath string, namespace string) {
	args := []string{"create", "-f", filePath, "--namespace", namespace}

	fmt.Println(fmt.Sprintf("Creating resource: %s", filePath))
	if output, err := exec.Command(cmdName, args...).CombinedOutput(); err != nil {
		fmt.Fprintf(os.Stderr, fmt.Sprintf("Failed to create resource %s: %v", filePath, string(output)))
	}
}

// Delete removes the given resource from the active kubernetes cluster
func Delete(filePath string, namespace string) {
	args := []string{"delete", "-f", filePath, "--namespace", namespace}

	if output, err := exec.Command(cmdName, args...).CombinedOutput(); err != nil {
		fmt.Fprintf(os.Stderr, fmt.Sprintf("Failed to delete resource %s: %v", filePath, string(output)))
	}
}

// WaitDeployReady blocks until the specified deployment has the required number of pods ready
func WaitDeployReady(resourceName string, minNumberReady int, namespace string) {
	waitResourceReady(resourceName, "deploy", "readyReplicas", minNumberReady, namespace)
}

// WaitDaemonSetReady blocks until the specified daemonset has the required number of pods ready
func WaitDaemonSetReady(resourceName string, minNumberReady int, namespace string) {
	waitResourceReady(resourceName, "daemonset", "numberReady", minNumberReady, namespace)
}

// WaitPodCompleted blocks until the specified pod is in the Succeeded phase
func WaitPodCompleted(podName string, namespace string) {
	fmt.Printf("Waiting for pod to finish running: %s\n", podName)
	args := []string{"get", "pod", podName, "-o", "jsonpath=\"{@.status.phase}\"", "--namespace", namespace}

	podCompleted := false
	for !podCompleted {
		time.Sleep(1000 * time.Millisecond)

		output, err := exec.Command(cmdName, args...).CombinedOutput()
		if err != nil {
			panic(fmt.Sprintf("Failed execute kubectl command to get pod phase: %v", string(output)))
		}

		phase := strings.Replace(string(output), "\"", "", -1)
		if strings.Compare(phase, "Succeeded") == 0 {
			podCompleted = true
		}
	}
}

func waitResourceReady(resourceName string, resourceType string, resourceCountField string, minNumberReady int, namespace string) {
	fmt.Printf("Waiting for readiness of %s %s\n", resourceType, resourceName)
	args := []string{"get", resourceType, resourceName, "-o", fmt.Sprintf("jsonpath=\"{@.status.%s}\"", resourceCountField), "--namespace", namespace}

	resourceReady := false
	for !resourceReady {
		time.Sleep(1000 * time.Millisecond)

		output, err := exec.Command(cmdName, args...).CombinedOutput()
		if err != nil {
			panic(fmt.Sprintf("Failed execute kubectl command to get deploy status: %v", string(output)))
		}

		readyReplicas, err := strconv.Atoi(strings.Replace(string(output), "\"", "", -1))
		if err == nil && readyReplicas >= minNumberReady {
			resourceReady = true
		}
	}
}
