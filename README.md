# kubecorr

Merge a pod's logs and the cluster events around it into one timeline.

When a pod fails, the app logs say one thing and `kubectl describe` says
another. kubecorr puts both in one list, sorted by time. You can see what the
cluster did (OOMKilled, BackOff, NodeNotReady) next to what the app printed.

## Install

```sh
go install github.com/dackota/kubecorr@latest
```

## Use

```sh
# one pod
kubecorr --context prod -n api api-7d9f-abc

# pods by label, last 30 minutes
kubecorr --context prod -n api -l app=api --since 30m

# logs of the crashed container, as JSON
kubecorr --context prod -n api api-7d9f-abc --previous -o json | jq .
```

`--context` is required. The tool never uses your current context by accident.

Output:

```
14:02:11.301 EVENT Warning BackOff    pod/api-7d9f-abc  Back-off restarting failed container
14:02:11.804 LOG   api-7d9f-abc/app  panic: runtime error: nil pointer dereference
14:02:15.000 EVENT Warning NodeNotReady node/node-1     Node node-1 status is now: NodeNotReady
```

## What it reads

- Logs: every container in the pod (init containers too), with kubelet
  timestamps. Use `-c NAME` for one container. Use `--previous` for the last
  crashed container.
- Events: for the pod, its owners (ReplicaSet, Deployment, Job, StatefulSet),
  and the node it runs on.

## Flags

| Flag | Default | Meaning |
|------|---------|---------|
| `--context` | (required) | kubeconfig context |
| `-n, --namespace` | from kubeconfig | namespace |
| `-l, --selector` | | pick pods by label instead of name |
| `-c, --container` | all | one container only |
| `-p, --previous` | false | logs of the last terminated container |
| `--since` | `1h` | how far back to look |
| `-o, --output` | `text` | `text` or `json` |

Times in text output are local time. JSON times are UTC.

Colors turn on when stdout is a terminal. Set `NO_COLOR=1` to turn them off.

## Limits

- Events live about one hour in most clusters. Older failures have logs but no
  events.
- Log times come from the kubelet, not from the app's own timestamp field.

## Develop

```sh
go test ./...
```

Tests use the fake client from `client-go`. No cluster is needed.
