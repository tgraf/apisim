{
    "kind":"ReplicationController",
    "apiVersion":"v1",
    "metadata":{
        "name":"status",
        "labels":{
            "k8s-app.apisim":"status"
        }
    },
    "spec":{
        "replicas":1,
        "selector":{
          "k8s-app.apisim":"status"
        },
        "template":{
            "metadata":{
                "labels":{
                   "k8s-app.apisim":"status"
                }
            },
            "spec":{
                "nodeSelector":{
                   "kubernetes.io/hostname": "worker2"
		},
                "containers":[{
                    "name":"status",
                    "image":"tgraf/apisim:latest",
		    "command": ["/go/bin/app", "status-server"],
                    "ports":[
                            {"containerPort": 8888, "name": "apisim-status"}
                    ]
                }]
            }
        }
    }
}
