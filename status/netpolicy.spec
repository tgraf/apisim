{
  "apiVersion": "extensions/v1beta1",
  "kind": "NetworkPolicy",
  "metadata": {
    "annotations": {
      "io.cilium.name": "k8s-app",
      "io.cilium.parent": "io.cilium.k8s"
    },
    "name": "policy-status"
  },
  "spec": {
    "podSelector": {
      "matchLabels": {
        "apisim":"status"
      }
    },
    "ingress": [
      {
        "from": [
          {
            "podSelector": {
              "matchLabels": {
                "io.cilium.reserved": "host"
              }
            }
          }
        ]
      }
    ]
  }
}
