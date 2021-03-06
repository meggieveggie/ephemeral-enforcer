package helpers

import (
	"flag"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

/*
GetConfig Gets the Kubernetes config from either your local (if you are running it locally)
or from the service account if you are running it in cluster
*/
func GetConfig() (*rest.Config, error) {
	var kubeconfig *string
	home := homeDir()
	kubeconfig = flag.String(
		"kubeconfig",
		filepath.Join(home, ".kube", "config"),
		"(optional) absolute path to the kubeconfig file",
	)
	flag.Parse()
	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		return nil, err
	}
	return config, nil
}

/*
GetClientSet Generates a clientset from the Kubeconfig
*/
func GetClientSet(kubeconfig *rest.Config) (kubernetes.Interface, error) {
	clientset, err := kubernetes.NewForConfig(kubeconfig)
	if err != nil {
		return nil, err
	}
	return clientset, nil
}

func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE") // windows
}

/*
GetEnv looks up an env key or returns a default
*/
func GetEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

/*
EphemeralChecks Struct that holds the information of the resources that could be deleted
*/
type EphemeralChecks struct {
	CreationTime metav1.Time
	Name         string
	Delete       bool
}

/*
RunChecks Run checks to see if resource should be deleted
*/
func (e *EphemeralChecks) RunChecks() {
	// Run The check to see if the age is past the TTL
	if passedTimeToLive(e.CreationTime) {
		e.Delete = true
	}
	// Run The Check for Skipped Names
	if !nameCheck(e.Name) {
		e.Delete = false
	}

}

func nameCheck(name string) bool {
	ephemeralEnforcerName := GetEnv("EPHEMERAL_ENFORCER_NAME", "ephemeral-enforcer")
	skippedPrefixes := GetEnv("SKIPPED_PREFIXES", "")
	if strings.Contains(name, ephemeralEnforcerName) {
		return false
	}
	for _, prefix := range strings.Split(skippedPrefixes, ",") {
		if strings.Contains(name, prefix) && prefix != "" {
			return false
		}
	}
	return true
}

func passedTimeToLive(creationTime metav1.Time) bool {
	ttl, _ := strconv.Atoi(os.Getenv("WORKLOAD_TTL"))
	// Convert to a negative
	ttl = 0 - ttl
	//
	previousTime := time.Now().Add(time.Minute * time.Duration(ttl))
	ttlTime := metav1.NewTime(previousTime)
	return creationTime.Before(&ttlTime)
}

/*
CheckDeleteResourceAllowed Checks if the resource is in the disallow list
*/
func CheckDeleteResourceAllowed(resourceType string) bool {
	disallowList := GetEnv("DISALLOW_LIST", "")
	for _, element := range strings.Split(disallowList, ",") {
		if strings.Contains(strings.ToLower(resourceType), strings.ToLower(element)) && element != "" {
			return false
		}
	}
	return true
}
