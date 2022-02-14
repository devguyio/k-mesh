// +build mage

package main

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
	"gopkg.in/yaml.v2"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/clientcmd"

	"knative.dev/kmesh/pkg/apis/mesh/v1alpha1"
	"knative.dev/kmesh/pkg/client/clientset/versioned"
	graphpint "knative.dev/kmesh/pkg/print"
)

// Default target to run when none is specified
// If not set, running mage will list available targets
// var Default = Build

const (
	BrokerCharts = "artifacts/brokers"
)

type ComponentState int64

const (
	Available ComponentState = iota
	Managed
	ManagedNotReady
	Bound
)

func (s ComponentState) String() string {
	switch s {
	case Managed:
		return text.FgGreen.Sprint("Managed")
	case ManagedNotReady:
		return text.FgYellow.Sprint("Managed (Not Ready)")
	case Bound:
		return text.Faint.Sprint("Bound (Not Managed)")
	case Available:
		return text.Faint.Sprint("Ready to install")
	}
	return "Unknown"
}

var kmesh = sh.OutCmd("kubectl", "-n", "knative-mesh")
var k = sh.OutCmd("kubectl")

type Brokers mg.Namespace

type Mesh mg.Namespace

type Infra mg.Namespace

type BrokerInfo struct {
	name,
	namespace,
	classes string
	state ComponentState
}

// Install Install Knative Mesh
func Install() error {
	if err := run("Installing Knative Eventing",
		exec.Command("kubectl", "apply", "-f", "artifacts/core/eventing-core.yaml")); err != nil {
		return err
	}
	if err := run("Installing Knative in-memory channel",
		exec.Command("kubectl", "apply", "-f", "artifacts/core/in-memory-channel.yaml")); err != nil {
		return err
	}
	return run("Installing Knative Mesh", exec.Command("kubectl", "apply", "-f",
		"artifacts/core/knative-mesh-operator.yaml"))
}

// Status Show the full status of all K-Mesh components
func Status() error {
	kmeshheader()
	fmt.Println(text.FormatUpper.Apply(text.Bold.Sprint("Kmesh Status:")))
	kmeshStatus()
	fmt.Println("\n")
	brokersStatus()
	return nil
}

// Build Build Knative Mesh
func (Infra) Build() error {
	cmd := exec.Command("bash", "-c", "KO_DOCKER_REPO=kind.local ko resolve -B --platform linux/amd64 -f config > artifacts/core/knative-mesh-operator.yaml")
	return run("Building", cmd)
}

// Kind Create a kind cluster for development purposes
func (Infra) Kind() error {

	return sh.Run("kind", "create", "cluster", "--config", "artifacts/conf/clusterconfig.yaml")
}

// InstallCLI Install kmesh cli to $GOPATH/bin
func (Infra) InstallCLI() error {
	if err := run("Building CLI binary", exec.Command("mage", "-compile", "kmesh")); err != nil {
		return err
	}
	bin := fmt.Sprintf("%s/bin", os.Getenv("GOPATH"))
	if err := run("Installing kmesh CLI to $GOPATH/bin", exec.Command("cp",
		"kmesh", bin)); err != nil {
		return err
	}
	return nil
}

// Start Start K-Mesh
func (Mesh) Start() error {
	kmeshheader()
	cmd := exec.Command("kubectl", "apply", "-f", "artifacts/core/kmesh.yaml")
	return run("Initializing Knative Mesh", cmd)
}

// Status Show the status of the K-Mesh
func (Mesh) Status() error {
	kmeshheader()
	return kmeshStatus()
}

// Status Show the brokers status
func (Brokers) Status() error {
	kmeshheader()
	return brokersStatus()
}

// List List the brokers status
func (Brokers) List() error {
	kmeshheader()
	return brokersStatus()
}

// Install Install an available Broker provider into Knative Mesh.
func (Brokers) Install(name string) error {
	brokersInfo := getBrokersInfo()
	if _, ok := brokersInfo[name]; !ok {
		printError(fmt.Sprintf("Can't find broker %s", name))
		return errors.New("failed to execute command")
	}
	binfo := brokersInfo[name]
	if binfo.state != Available {
		printError(fmt.Sprintf("Broker %s is already installed.", name))
		return errors.New("failed to execute command")
	}
	brokerChart := fmt.Sprintf("%s/%s", BrokerCharts, name)
	cmd := exec.Command("helm", "install", name, brokerChart, "--namespace", "knative-eventing")
	if err := run("Installing broker "+name, cmd); err != nil {
		printError(fmt.Sprintf("The following error occured when installing broker %s\nError: %v", name, err))
		return errors.New(fmt.Sprintf("installing broker %s failed", name))
	}
	return nil
}

func brokersStatus() error {
	fmt.Println(text.FormatUpper.Apply(text.Bold.Sprint("Broker Providers:")))
	t := newTable()
	addBrokersRows(t)
	t.Render()
	return nil
}

func kmeshStatus() error {
	t := newTable()
	t.SetColumnConfigs([]table.ColumnConfig{
		{
			Number:    1,
			AutoMerge: true,
		},
	})
	if err := addMeshRows(t); err != nil {
		return err
	}
	t.Render()
	return nil
}

func addMeshRows(t table.Writer) error {
	t.AppendHeader(table.Row{"Broker Classes", "Ingress Details", "Status"})
	clientSet, err := newClientSet()
	if err != nil {
		return err
	}
	kmeshList, err := clientSet.MeshV1alpha1().KMeshes("knative-mesh").List(context.TODO(), v1.ListOptions{})
	if err != nil {
		return err
	}
	if len(kmeshList.Items) > 0 {
		for _, m := range kmeshList.Items {
			t.AppendRows(newMeshTableRow(&m))
		}
	} else {
		t.AppendRows(newMeshTableRow(nil))
	}
	return nil
}

func newMeshTableRow(mesh *v1alpha1.KMesh) []table.Row {
	if mesh == nil {
		return []table.Row{
			{"-", "-", text.FgYellow.Sprint("Not initialized")},
		}
	}
	rows := make([]table.Row, 0, 0)
	classes := mesh.Status.BrokerClasses
	if len(classes) == 0 {
		classes = []string{"-"}
	}
	var ingresses = getIngressEntries(mesh.Status.Ingresses)

	if len(ingresses) > len(classes) {
		c := 0
		last := len(classes) - 1
		for _, i := range ingresses {
			rows = append(rows, table.Row{classes[c], i, text.FgGreen.Sprint("Ready")})
			if c < last {
				c++
			}
		}
	} else {
		i := 0
		last := len(ingresses) - 1
		for _, c := range classes {
			rows = append(rows, table.Row{c, ingresses[i], text.FgGreen.Sprint("Ready")})
			if i < last {
				i++
			}
		}
	}
	return rows
}

func getIngressEntries(ingresses v1alpha1.Ingresses) []string {

	if len(ingresses) == 0 {
		return []string{"-"}
	}
	result := make([]string, 0, len(ingresses))
	for _, i := range ingresses {
		l := fmt.Sprintf("Broker: %s \nURL: %s\n", i.Name, i.Address.URL)
		if len(i.Egresses) > 0 {
			l = fmt.Sprintf("%s\nTopology:\n", l)
			n := graphpint.NewNode(fmt.Sprintf("[B] %s", i.Name))
			for _, e := range i.Egresses {
				n.Insert(fmt.Sprintf("[%s/%s]", e.Destination.Ref.Namespace, e.Destination.Ref.Name))
			}
			l = fmt.Sprintf("%s%s\n", l, graphpint.SPPrint(n))
		}

		result = append(result, l)
	}
	return result
}

func newClientSet() (*versioned.Clientset, error) {
	kubeconfig := filepath.Join(
		os.Getenv("HOME"), ".kube", "config",
	)
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return nil, err
	}
	clientSet, err := versioned.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	return clientSet, nil
}

func newTable() table.Writer {
	t := table.NewWriter()
	t.SetStyle(table.StyleLight)
	t.Style().Color.Header = text.Colors{text.Bold}
	t.SetOutputMirror(os.Stdout)
	return t
}

func addBrokersRows(t table.Writer) error {
	//rowConfigAutoMerge := table.RowConfig{AutoMerge: true}
	//t.AppendHeader(table.Row{"Knative Mesh", "Knative Mesh", "Knative Mesh", "Knative Mesh"}, rowConfigAutoMerge)
	t.AppendHeader(table.Row{"Broker", "Broker Class", "Version", "Status"})

	brokers := getBrokersInfo()
	for _, b := range brokers {
		t.AppendRow(newBrokerTableRow(b))
	}
	return nil
}

func getBrokersInfo() map[string]BrokerInfo {
	items, err := os.ReadDir(BrokerCharts)
	brokers := make(map[string]BrokerInfo)
	if err != nil {
		return nil
	}

	for _, i := range items {
		if i.IsDir() {
			name := i.Name()
			binfo := BrokerInfo{name: name, state: Available}
			binding := fmt.Sprintf("artifacts/brokers/%s/templates/mesh-brokerbinding.yaml", i.Name())
			yamlFile, err := ioutil.ReadFile(binding)
			if err != nil {
				log.Fatalf("yamlFile.Get err   #%v ", err)
				return nil
			}
			bb := v1alpha1.BrokerBinding{}
			err = yaml.Unmarshal(yamlFile, &bb)
			if err != nil {
				log.Fatalf("Unmarshal: %v", err)
				return nil
			}
			classes := ""
			lead := ""
			for _, c := range bb.Spec.Classes {
				classes = fmt.Sprintf("%s%s%s", lead, classes, c)
				lead = "\n"
			}
			binfo.classes = classes
			brokers[name] = binfo
		}
	}
	out, err := k("get", "-o", "jsonpath={range .items[*]}{.metadata.name} {.metadata.namespace} {.status.conditions[?(@.type==\"Ready\")].status} {.spec.classes}{\"\\n\"}{end}", "-A", "brokerbindings.mesh.knative.dev")
	if err != nil {
		return nil
	}
	for _, l := range strings.Split(out, "\n") {
		if l := strings.TrimSpace(l); l != "" {
			binding := strings.Fields(l)
			name := binding[0]
			namespace := binding[1]
			ready := binding[2]
			classes := strings.ReplaceAll(binding[3], "\"", "")
			classes = strings.ReplaceAll(classes, "[", "")
			classes = strings.ReplaceAll(classes, "]", "")
			if binfo, ok := brokers[name]; !ok {
				brokers[name] = BrokerInfo{
					name:      name,
					namespace: namespace,
					state:     Bound,
					classes:   classes,
				}
			} else {
				binfo.namespace = namespace
				binfo.classes = classes
				if ready == "True" {
					binfo.state = Managed
				} else {
					binfo.state = ManagedNotReady
				}
				brokers[name] = binfo
			}
		}
	}
	return brokers
}

func newBrokerTableRow(broker BrokerInfo) table.Row {
	name := broker.name
	if broker.state != Available {
		name = fmt.Sprintf("%s/%s", broker.namespace, broker.name)
	}
	return table.Row{name, broker.classes, "1.2.0", broker.state.String()}
}

func run(prompt string, cmd *exec.Cmd) error {
	fmt.Printf("%s..........", prompt)
	return doRun(cmd)
}

func doRun(cmd *exec.Cmd) error {
	err := cmd.Run()
	if err != nil {
		return err
	}
	fmt.Println(text.FgGreen.Sprint("Done"))
	return nil
}

func printError(msg string) {
	fmt.Printf("%s\t%s\n", text.FgRed.Sprintf("ERROR:"), msg)
}

func kmeshheader() {
	header := text.Bold.Sprintf("Knative Mesh 1.2.0")
	fmt.Printf("\n%s\n\n", header)
}
