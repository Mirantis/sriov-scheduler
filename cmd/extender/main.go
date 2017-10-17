package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/dshulyak/sriov-scheduler/pkg/extender"
	"github.com/spf13/pflag"
)

type options struct {
	listen           string
	kubeconfig       string
	promisesInterval time.Duration
}

func (o *options) register() {
	pflag.StringVarP(&o.listen, "listen", "l", ":8989", "Socket to listen on.")
	pflag.StringVar(&o.kubeconfig, "kubeconfig", "", "Kubernetes config file.")
	pflag.DurationVarP(
		&o.promisesInterval, "promises-interval", "p", 10*time.Second,
		"Defines how long SR-IOV VFs will be promised to a particular pod.")
}

func (o *options) parse() {
	pflag.Parse()
}

func (o *options) registerAndParse() {
	o.register()
	o.parse()
}

func main() {
	log.SetOutput(os.Stderr)
	opts := new(options)
	opts.registerAndParse()
	log.Printf("Using kubernetes config %s\n", opts.kubeconfig)
	config, err := clientcmd.BuildConfigFromFlags("", opts.kubeconfig)
	if err != nil {
		log.Fatal(err)
	}
	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatal(err)
	}
	stopCh := make(chan struct{})
	ext := extender.NewExtender(client)
	ctl := ext.CreateMonitor()
	go func() {
		ctl.Run(stopCh)
	}()
	log.Println("wait until controller and cache synced with api server")
	if err := wait.PollImmediate(1*time.Second, 10*time.Second, func() (bool, error) {
		return ctl.HasSynced(), nil
	}); err != nil {
		log.Fatalf("error waiting for a controller to sync with api server: %v", err)
	} else {
		fmt.Println("controller synced with api server successfully")
	}
	go func() {
		ext.RunPromisesCleaner(opts.promisesInterval, stopCh)
	}()
	srv := extender.MakeServer(ext, opts.listen)
	log.Fatal(srv.ListenAndServe())
}
