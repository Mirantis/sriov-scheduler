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

	"time"

	"strings"

	"github.com/spf13/pflag"
)

type options struct {
	device     string
	kubeconfig string
	interval   time.Duration
	nodename   string
}

func (o *options) register() {
	pflag.StringVar(&o.device, "device", "eth0", "Device to use for VFs.")
	pflag.StringVar(&o.kubeconfig, "kubeconfig", "", "Kubernetes config file.")
	pflag.DurationVarP(&o.interval, "interval", "i", 0, "If set discovery will run every specified interval.")
	hostname, err := os.Hostname()
	if err != nil {
		log.Fatalf("Error getting node hostname: %v", err)
	}
	pflag.StringVar(&o.nodename, "nodename", hostname, "Name of the node.")
}

func (o *options) parse() {
	pflag.Parse()
}

func (o *options) registerAndParse() {
	o.register()
	o.parse()
}

const (
	sriovTotalvfsMask                 = "/test/class/net/%s/device/sriov_totalvfs"
	TotalVFsResource  v1.ResourceName = "totalvfs"
)

func main() {
	log.SetOutput(os.Stderr)

	opts := new(options)
	opts.registerAndParse()
	err := periodically(opts.interval, func() error {
		deviceFile := fmt.Sprintf(sriovTotalvfsMask, opts.device)
		log.Printf("Total VFs number will be discovered from %s\n", deviceFile)
		totalVfsBytes, err := ioutil.ReadFile(deviceFile)
		if err != nil {
			log.Fatalf("Error discovering totalvfs from file %s; %v", deviceFile, err)
		}
		totalVfs := resource.MustParse(strings.TrimSpace(string(totalVfsBytes)))
		log.Printf("Using kubernetes config %s\n", opts.kubeconfig)
		config, err := clientcmd.BuildConfigFromFlags("", opts.kubeconfig)
		if err != nil {
			log.Fatal(err)
		}
		client, err := kubernetes.NewForConfig(config)
		if err != nil {
			log.Fatal(err)
		}
		if err := doDiscovery(opts.nodename, totalVfs, client); err != nil {
			log.Fatalf("Error updating totalvfs for a node %s: %v\n", opts.nodename, err)
		}
		return nil
	})
	if err != nil {
		log.Fatal(err)
	}
	os.Exit(0)
}

func doDiscovery(hostname string, totalVfs resource.Quantity, client *kubernetes.Clientset) error {
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
		_, err = client.Nodes().UpdateStatus(node)
		if err != nil {
			log.Printf("Updating a node %s failed.\n", hostname)
			continue
		}
		return nil
	}
	return nil
}

func periodically(interval time.Duration, f func() error) error {
	for {
		if err := f(); err != nil {
			return err
		}
		if interval > 0 {
			time.Sleep(interval)
			continue
		}
		return nil
	}
}
