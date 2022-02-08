// +build mage

package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
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

// Build Build Knative Mesh
func Build() error {
	cmd := exec.Command("bash", "-c", "KO_DOCKER_REPO=kind.local ko resolve --platform linux/amd64 -f config > artifacts/knative-mesh-operator.yaml")
	return run("Building", cmd)
}

func InstallEventing() error {
	cmd := exec.Command("kubectl", "apply", "-f", "artifacts/eventing-core.yaml")
	err := run("Installing Knative Eventing", cmd)
	if err != nil {
		return err
	}
	cmd = exec.Command("kubectl", "apply", "-f", "artifacts/in-memory-channel.yaml")
	return run("Installing Knative in-memory channel", cmd)
}

// Install Install Knative Mesh
func Install() error {
	mg.Deps(InstallEventing)
	cmd := exec.Command("kubectl", "apply", "-f", "artifacts/knative-mesh-operator.yaml")
	return run("Installing Knative Mesh", cmd)
}

func Init() error {
	kmeshheader()
	cmd := exec.Command("kubectl", "apply", "-f", "artifacts/kmesh.yaml")
	return run("Initializing Knative Mesh", cmd)
}

type Brokers mg.Namespace
type Mesh mg.Namespace

func (Mesh) List() error {
	return nil
}

func (Brokers) List() {
	t := table.NewWriter()
	addBrokers(t)
	t.Render()
}

// Install Install an available Broker provider into Knative Mesh.
func (Brokers) Install(name string) error {
	brokersInfo := getBrokers()
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

type BrokerInfo struct {
	name,
	namespace,
	classes string
	state ComponentState
}

func getBrokers() map[string]BrokerInfo {
	items, err := os.ReadDir(BrokerCharts)
	brokers := make(map[string]BrokerInfo)
	if err != nil {
		return nil
	}

	for _, i := range items {
		if i.IsDir() {
			name := i.Name()
			brokers[name] = BrokerInfo{name: name, state: Available}
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
			classes := binding[3]
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

func printError(msg string) {
	fmt.Printf("%s\t%s\n", text.FgRed.Sprintf("ERROR:"), msg)
}
func addBrokers(t table.Writer) error {
	rowConfigAutoMerge := table.RowConfig{AutoMerge: true}
	brokers := getBrokers()
	t.SetStyle(table.StyleLight)
	t.Style().Color.Header = text.Colors{text.Bold}
	t.SetOutputMirror(os.Stdout)
	t.AppendHeader(table.Row{"Knative Mesh", "Knative Mesh", "Knative Mesh", "Knative Mesh"}, rowConfigAutoMerge)
	t.AppendHeader(table.Row{"Broker", "Broker Class", "Version", "Status"})

	for _, b := range brokers {
		t.AppendRow(brokerTableRow(b))
	}
	return nil
}
func BrokersStatus() error {
	kmeshheader()
	return brokersStatus()
}

func brokersStatus() error {
	t := table.NewWriter()
	addBrokers(t)
	t.Render()
	return nil
}

func Status() error {
	kmeshheader()
	brokersStatus()
	//kmeshStatus()
	return nil
}

func brokerTableRow(broker BrokerInfo) table.Row {
	name := broker.name
	if broker.state != Available {
		name = fmt.Sprintf("%s/%s", broker.namespace, broker.name)
	}
	return table.Row{name, "MTChannelBasedBroker", "1.2.0", broker.state.String()}
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

func kmeshheader() {
	header := text.Bold.Sprintf("Knative Mesh 1.2.0")
	fmt.Printf("\n%s\n\n", header)
}
