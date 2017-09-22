package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"k8s.io/apimachinery/pkg/api/resource"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/spf13/pflag"
)

type options struct {
	device     string
	kubeconfig string
}

func (o *options) register() {
	pflag.StringVar(&o.device, "device", "eth0", "Device to use for VFs.")
	pflag.StringVar(&o.kubeconfig, "kubeconfig", "", "Kubernetes config file.")
}

func (o *options) parse() {
	pflag.Parse()
}

func (o *options) registerAndParse() {
	o.register()
	o.parse()
}

const (
	sriovTotalvfsMask                 = "/sys/class/net/%s/device/sriov_totalvfs"
	TotalVFsResource  v1.ResourceName = "totalvfs"
)

func main() {
	log.SetOutput(os.Stderr)

	opts := new(options)
	opts.registerAndParse()
	deviceFile := fmt.Sprintf(sriovTotalvfsMask, opts.device)
	log.Printf("Total VFs number will be discovered from %s\n", deviceFile)
	totalVfsBytes, err := ioutil.ReadFile(deviceFile)
	if err != nil {
		log.Fatalf("Error discovering totalvfs from file %s; %v", deviceFile, err)
	}
	totalVfs := resource.MustParse(string(totalVfsBytes))

	log.Printf("Using kubernetes config %s\n", opts.kubeconfig)
	config, err := clientcmd.BuildConfigFromFlags("", opts.kubeconfig)
	if err != nil {
		log.Fatal(err)
	}
	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatal(err)
	}

	hostname, err := os.Hostname()
	if err != nil {
		log.Fatalf("Error getting node hostname: %v", err)
	}

	for i := 3; i > 0; i-- {
		log.Printf("Fetching a node %s from kubernetes API. Retries left %d\n", hostname, i-1)
		node, err := client.Nodes().Get(hostname, meta_v1.GetOptions{})
		if err != nil {
			log.Printf("Getting a node %s failed.\n", hostname)
			continue
		}
		log.Printf("Updating a node %s with totalvfs %s\n", hostname, totalVfs)
		// TODO a patch request
		node.Status.Capacity[TotalVFsResource] = totalVfs
		node.Status.Allocatable[TotalVFsResource] = totalVfs
		_, err = client.Nodes().Update(node)
		if err != nil {
			log.Printf("Updating a node %s failed.\n", hostname)
			continue
		}
		os.Exit(0)
	}
	log.Fatalf("Not able to update totalvfs resource a node %s\n", hostname)
}
