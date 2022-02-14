# K-Mesh

K-Mesh is an experimental Knative distribution which provides a fresh, CLI-focused, holistic user experience of running and managing Knative.

> **_NOTE:_** K-Mesh is an early-stage PoC project.

## Demo
>### Prerequisites
>
>- [Kind](https://kind.sigs.k8s.io/)
>- [Mage](https://magefile.org/) (For development only)

Clone K-Mesh repo

```bash
git clone git@github.com:devguyio/kmesh.git
```

Change to the demo directory

```bash
cd kmesh/demo
```
Copy the `bin/kmesh` binary to your PATH or execute the kmesh commands from inside the demo directory (i.e. `./bin/kmesh`).

> **_NOTE:_** Currently some paths are hardcoded, so you always need to be in the demo directory (which contains the `artifacts` directory) when issuing `kmesh` commands.

Create a Kind cluster using `kmesh`

```bash
kmesh infra:kind
```

Use `kmesh` to install Knative. Currently this will install:
- Knative Eventing core
- InMemoryChannel
- K-Mesh operator

```bash
kmesh install
```

Verify that the K-Mesh is now showing `Not initialized` when checking the status
```bash
kmesh mesh:status
```

Start the default K-Mesh

```bash
kmesh mesh:start
```
Verify that the K-Mesh is now showing `Ready` when checking the status

```bash
kmesh mesh:status
```
List available brokers. Kmesh demo comes with the `mtbroker` and two demo brokers under `artifacts/brokers`.

```bash
kmesh brokers:status
```
Install the `mtbroker`. K-Mesh uses Helm charts to manage the K-Mesh compontents.

```bash
kmesh brokers:install mtbroker
```

Verify that the K-Mesh is now showing `Ready` when checking the brokers status. You can use the top level `status` command to show the full K-Mesh status including brokers. Notice how the default K-Mesh is now aware of the `MTChannelBasedBroker` in the `BROKER CLASSES` column.

```bash
kmesh status
```
Now it's time to create some brokers and triggers. The demo directory comes with some demo manifests. Follow these steps to explore the K-Mesh features.

```bash
# Create one broker in the dev namespace 
kubectl apply -f 1-broker-dev.yaml

# Let's see if the K-Mesh sees the new ingress
kmesh status

# Create a trigger & subscriber
kubectl apply 2-trigger-dev.yaml

# Verify that the egress is listed in the status
kmesh status

# Create some extra triggers & subscripbers
kubectl apply 3-trigger-dev.yaml

# Verify that the egresses
kmesh status

# Full annihilation 
kubectl apply 4-large-demo.yaml

# Enjoy the view!
kmesh status

```

## TODO

- Mesh controller PoC
    - BrokerImpl -> Kmesh Classes
    - New broker -> Kmesh Ingress
    - New trigger -> Kmesh Egress
- Kafka-Broker <-> Controller
  - Install K-Mesh dest
- Helm chart in the broker-binding
- Get brokerclasses from helm chart
