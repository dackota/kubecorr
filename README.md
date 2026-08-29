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
# every pod in a namespace
kubecorr -n api

# one pod
kubecorr --context prod -n api api-7d9f-abc

# pods by label, last 30 minutes
kubecorr --context prod -n api -l app=api --since 30m

# follow live, like kubectl logs -f, but with events too
kubecorr -n api -f

# only failure lines
kubecorr -n api --grep "panic|timeout|OOM"

# logs of the crashed container, as JSON
kubecorr --context prod -n api api-7d9f-abc --previous -o json | jq .
```

Output:

```
api-7d9f-abc  node=node-1  phase=Running
  app                  ready=false restarts=3   Waiting: CrashLoopBackOff  last exit: OOMKilled (137)

14:02:11.301 EVENT Warning BackOff    pod/api-7d9f-abc  Back-off restarting failed container
14:02:11.804 LOG   api-7d9f-abc/app  panic: runtime error: nil pointer dereference
14:02:15.000 EVENT Warning NodeNotReady node/node-1     Node node-1 status is now: NodeNotReady
```

## What it reads

- Summary header: each container's restart count, state, and last exit code
  and reason. Also probe failures per type (readiness, liveness, startup)
  with a count and the time of the last one.
- Logs: every container in the pod (init containers too), with kubelet
  timestamps. Use `-c NAME` for one container. Use `--previous` for the last
  crashed container.
- Events: for the pod, its owners (ReplicaSet, Deployment, Job, StatefulSet),
  and the node it runs on. Node events live in the `default` namespace and
  are still found. Events about other pods in other namespaces are not, unless
  you add `--extra-ns kube-system` (Warnings only, to keep the noise down).
- Status items (type `Status`): pod conditions, container exits, and node
  trouble (MemoryPressure, DiskPressure, NotReady). These come from the
  object status, so they are still there after Events expire.
- Log lines with words like error, panic, fatal, OOM, timeout print in red.

## Flags

| Flag | Default | Meaning |
|------|---------|---------|
| `--context` | current context | kubeconfig context |
| `-n, --namespace` | from kubeconfig | namespace |
| `-l, --selector` | all pods | pick pods by label |
| `-c, --container` | all | one container only |
| `-p, --previous` | false | logs of the last terminated container |
| `--since` | `1h` | how far back to look |
| `--tail` | `1000` | max log lines per container; `0` for no limit |
| `-o, --output` | `text` | `text` or `json` |
| `-t, --tui` | false | side by side view, see below |
| `-f, --follow` | false | keep running, show new logs and events live |
| `--grep` | | keep only log lines that match this regex; events always stay |
| `--extra-ns` | | also show Warning events from these namespaces, tagged `ns/kind/name` |

Times in text output are local time. JSON times are UTC.

Colors turn on when stdout is a terminal. Set `NO_COLOR=1` to turn them off.

## Side by side view

```sh
kubecorr -n api --tui
```

Logs on the left, events on the right. The two panes are linked by time.
When you scroll one, the other jumps to the nearest item at or before that
time.

| Key | Action |
|-----|--------|
| `j` `k` or arrows | scroll one line |
| `ctrl+d` `ctrl+u` or PgDn PgUp | scroll ten lines |
| `g` `G` | top, bottom |
| `tab` | switch pane |
| `w` | toggle line wrap |
| `/` | type a filter, then enter. Filters both panes by pod, reason, or text. `esc` clears |
| `f` | toggle follow (auto scroll to newest, on by default with `-f`) |

With `-f`, the header reloads every 3 seconds.
| `q` | quit |

## Limits

- Events live about one hour in most clusters. Older failures have logs but no
  events.
- Log times come from the kubelet, not from the app's own timestamp field.
- In `-f` text mode, lines print as they arrive. Items from different pods
  can be a little out of time order. The `--tui` view keeps them sorted.

## Develop

```sh
go test ./...
```

Tests use the fake client from `client-go`. No cluster is needed.

To check the tool against a real cluster, deploy the broken apps in
`testdata/broken.yaml`. One crash loops with failing probes, one gets OOM
killed. Then run the tool on the `kubecorr-test` namespace.

```sh
kubectl --context <ctx> apply -f testdata/broken.yaml
sleep 90
kubecorr --context <ctx> -n kubecorr-test
kubecorr --context <ctx> -n kubecorr-test --tui -f
kubectl --context <ctx> delete namespace kubecorr-test
```
