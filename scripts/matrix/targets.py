"""A matrix target: one dgem serving base URL.

  Vertex dedicated endpoint: https://<ID>.<REGION>-<NUM>.prediction.vertexai.goog/v1/projects/<P>/locations/<R>/endpoints/<ID>/invoke
  Cloud Run:                 https://<service>-<hash>-<region>.run.app   (a tagged revision URL works too)
  Self-hosted:               http://<GPU_HOST>:8080
"""
import json

from . import net


class Target:
    def __init__(self, name, url):
        self.name, self.base = name, url.rstrip("/")
        if self.base.endswith("/v1"):
            self.base = self.base[:-3]

    @property
    def kind(self):
        if net.is_vertex(self.base):
            return "vertex"
        if ".run.app" in self.base:
            return "cloudrun"
        return "self-hosted"

    @property
    def cli_url(self):
        """What `dgem -u` expects: the invoke base for Vertex (the client appends the route), <base>/v1 otherwise."""
        return self.base if self.kind == "vertex" else self.base + "/v1"

    def health(self):
        st, body, _, _ = net.request(self.base + "/health", timeout=60, retries=1)
        return st, body

    def systemone(self, body, timeout=300):
        return net.request(self.base + "/v1/systemone", body, timeout=timeout)

    def chat(self, schema, state, timeout=300):
        payload = {"model": "dgemma", "messages": [{"role": "system", "content": json.dumps(schema)},
                                                   {"role": "user", "content": json.dumps(state, ensure_ascii=False)}],
                   "logprobs": True, "top_logprobs": 5}
        return net.request(self.base + "/v1/chat/completions", payload, timeout=timeout)

    # scripts/contract_diff.py calls t.request(path, body, method) -> (status, raw bytes, headers)
    def request(self, path, body=None, method="POST"):
        st, b, _, h = net.request(self.base + path, body, method=method, retries=0, raw=True)
        if isinstance(b, dict):  # network error
            b = json.dumps(b).encode()
        return st, b, h

    def redacted(self):
        return {"name": self.name, "kind": self.kind}
