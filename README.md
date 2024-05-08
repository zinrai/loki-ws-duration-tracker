# loki-ws-duration-tracker

This programme is a sampled code that uses input values to generate Loki search labels and uses the Loki WebSocket API to retrieve logs for a given pod.

This program is designed to retrieve logs from Loki for a given pod and measure the time it takes for the logs to become available in Loki after the pod has started. It reads input from stdin in JSON format, including the pod start time, task run name, and target namespace.

**The program uses the tail endpoint provided by the Loki HTTP API as a WebSocket.**

## Futures

This programme must be run at the same time as the k8s Pod is started.

If you know of a better way to do this, please let me know.

## Motivation

I wanted to measure the time it took for the logs to become searchable at Loki.

## Tested Version

- `Loki`: 2.0.0
    - https://grafana.com/docs/loki/v2.0.x/api/#get-lokiapiv1tail

## Requirements

Access to a Grafana Loki instance with `auth_enabled: true`

https://grafana.com/docs/loki/latest/configure/#supported-contents-and-default-values-of-lokiyaml

## Usage

If Loki is deployed on k8s, port forward to the Loki search endpoint.

```
$ kubectl port-forward svc/querier 3100:3100 -n loki
Forwarding from 127.0.0.1:3100 -> 3100
Forwarding from [::1]:3100 -> 3100
```

Prepare a `config.yaml` file with the following structure:

* `loki_address`: The base URL of your Loki instance.
* `loki_websocket_address`: The WebSocket URL of your Loki instance.
* `loki_label_key`: The label key used to identify pods in Loki.

```bash
$ cat << EOF > config.yaml
loki_address: "http://localhost:3100"
loki_websocket_address: "ws://localhost:3100"
loki_label_key: "my-task-run"
EOF
```

Run the program and provide the required input via stdin in JSON format:

```bash
$ cat << EOF > input.json
{ "podStartTime": "2023-05-08T12:34:56Z", "taskRunName": "my-task-run", "targetNamespace": "my-namespace" }
EOF
```

```bash
$ go run main.go < input.json
2023/05/08 15:00:00 First log line for pod my-task-run in namespace my-namespace: (Time difference: 4.239230187s)
```

If an error occurs during log retrieval or no logs are found within 1 minute, the program will print an error message and exit.

```
$ go run main.go < input.json
2023/05/08 15:00:00 Error getting logs for pod my-task-run in namespace my-namespace: failed to get logs for pod my-task-run after 1 minute
```

## Example

Example when used in combination with [tekton-task-run-creator](https://github.com/zinrai/tekton-task-run-creator) :

```bash
../tekton-task-run-creator/tekton-task-run-creator | jq -c -R 'fromjson? | select(type == "object")' | ./loki-ws-duration-tracker
2023/05/08 15:00:00 First log line for pod my-task-run in namespace my-namespace: (Time difference: 4.239230187s)
```

## License

This project is licensed under the MIT License - see the [LICENSE](https://opensource.org/license/mit) for details.
