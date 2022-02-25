package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
)

const (
	configMapNameDefault = "pts"
	ptsResultsDir        = "/var/lib/phoronix-test-suite/test-results/"
	ptsResultsFile       = "composite.xml"
)

func getConfigMapName() string {
	name := os.Getenv("POD_NAME")
	if len(name) != 0 {
		return name
	}
	klog.Errorf("POD_NAME unset or empty")

	// Fallback to HOSTNAME.
	name = os.Getenv("HOSTNAME")
	if len(name) != 0 {
		return name
	}
	klog.Errorf("HOSTNAME unset or empty")

	// We really need some ConfigMap name.
	now := time.Now()
	return now.Format("2006-01-02T15.04.05")
}

func findFile(dir, filename string) (string, error) {
	resultsFile := ""

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && info.Name() == filename {
			resultsFile = path
			return nil
		}
		return nil
	})

	return resultsFile, err
}

func getFileContents(file string) (string, error) {
	content, err := ioutil.ReadFile(file)
	return string(content), err
}

func cmList(clientset *kubernetes.Clientset) {
	namespace := ""
	cms, err := clientset.CoreV1().ConfigMaps(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		klog.Errorf(err.Error())
	}

	for _, cm := range cms.Items {
		fmt.Printf("%s %s\n", cm.GetName(), cm.GetCreationTimestamp())
	}
}

func cmWrite(clientset *kubernetes.Clientset) error {
	cmName := getConfigMapName()

	fileResults, err := findFile(ptsResultsDir, ptsResultsFile)
	if err != nil {
		return fmt.Errorf("error searching for results file %q in %q: %v\n", ptsResultsDir, ptsResultsFile, err)
	}

	cmData, err := getFileContents(fileResults)
	if err != nil {
		return fmt.Errorf("failed to read contents of file %q: %v", fileResults, err)
	}

	namespace := os.Getenv("WATCH_NAMESPACE")
	if len(namespace) == 0 {
		namespace = metav1.NamespaceDefault
	}

	configMap := corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{Namespace: namespace, Name: cmName},
		Data:       map[string]string{"composite.xml": string(cmData)},
	}

	_, err = clientset.CoreV1().ConfigMaps(namespace).Update(context.TODO(), &configMap, metav1.UpdateOptions{})
	if err == nil {
		return nil
	}
	// There was an error updating the ConfigMap.  Deal with it.
	if apierrors.IsNotFound(err) {
		// We have not created the ConfigMap yet.
		_, err = clientset.CoreV1().ConfigMaps(namespace).Create(context.TODO(), &configMap, metav1.CreateOptions{})
		if err == nil {
			return nil
		}
	}

	if err != nil {
		return fmt.Errorf("failed to write results ConfigMap: %v", err)
	}

	return nil
}

func main() {
	config, err := GetConfig()
	if err != nil {
		klog.Fatalf("unable to get k8s config: %v", err)
	}

	client := kubernetes.NewForConfigOrDie(config)
	_ = client
	err = cmWrite(client)
	if err != nil {
		klog.Fatalf("failed to write ConfigMap with results: %v", err)
	}
}
