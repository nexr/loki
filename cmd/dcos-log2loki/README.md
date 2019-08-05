DC/OS Log to loki
=================

Current status: `Proof of Concept`

# build
```
$ cd cmd/dcos-log2loki
$ go build
```

# start

### deploy loki on DC/OS

```json
{
  "labels": {},
  "id": "/test/loki",
  "cmd": "/usr/bin/loki -config.file=/etc/loki/local-config.yaml",
  "container": {
    "portMappings": [
      {
        "containerPort": 3100,
        "labels": {
          "VIP_0": "api.test/loki:3100"
        },
        "servicePort": 0,
        "name": "api"
      }
    ],
    "type": "DOCKER",
    "volumes": [],
    "docker": {
      "image": "grafana/loki:latest",
      "forcePullImage": true,
      "privileged": false,
      "parameters": []
    }
  },
  "cpus": 1,
  "disk": 0,
  "healthChecks": [
    {
      "gracePeriodSeconds": 300,
      "intervalSeconds": 60,
      "maxConsecutiveFailures": 3,
      "portIndex": 0,
      "timeoutSeconds": 20,
      "delaySeconds": 15,
      "protocol": "MESOS_HTTP",
      "path": "/ready",
      "ipProtocol": "IPv4"
    }
  ],
  "instances": 1,
  "maxLaunchDelaySeconds": 3600,
  "mem": 2048,
  "gpus": 0,
  "networks": [
    {
      "name": "dcos",
      "mode": "container"
    }
  ],
  "requirePorts": false,
  "upgradeStrategy": {
    "maximumOverCapacity": 1,
    "minimumHealthCapacity": 1
  },
  "killSelection": "YOUNGEST_FIRST",
  "unreachableStrategy": {
    "inactiveAfterSeconds": 0,
    "expungeAfterSeconds": 0
  },
  "fetch": [],
  "constraints": []
}
```

### start log2loki on each node

```
$ dcos-log2loki
```
