# Copyright 2026 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

"""A matrix target: one dgem serving base URL.

  Vertex dedicated endpoint: https://<ID>.<REGION>-<NUM>.prediction.vertexai.goog/v1/projects/<P>/locations/<R>/endpoints/<ID>/invoke
  Cloud Run:                 https://<service>-<hash>-<region>.run.app   (a tagged revision URL works too)
  Self-hosted:               http://<GPU_HOST>:8080

A target may carry request options after "#", for an A/B of an opt-in serving option on one deployment:
  --target docfirst=<url>#layout=document_first
They are added to every /v1/systemone and chat schema body, and the local adapter gets the matching flag
(layout -> --prompt-layout). `dgem decide` suites (templates) do not get them.

A competitor (another decision model with the Jev /v1/systemone contract) is a CompetitorTarget:
  --competitor strands=<url>#profile=strands-decider-2b
It runs only the /v1/systemone suites (and latency), directly (never through the dgem adapter), never gets dgem-only
body fields, and is excluded from every verdict gate. The profile (benchmarks/competitors/<profile>.json) supplies the
`model` field, the option limit, and the training-exposure map used by the comparison report.
"""
import json
import os

from . import net


class Target:
    OPTIONS = {"layout": ("schema_first", "document_first")}
    role = "dgem"

    def __init__(self, name, url):
        url, _, frag = url.partition("#")
        self.options = dict(kv.split("=", 1) for kv in frag.split("&") if kv)
        for k, v in self.options.items():
            if v not in self.OPTIONS.get(k, ()):
                raise SystemExit(f"target {name}: unknown option {k}={v} (known: {self.OPTIONS})")
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
        return net.request(self.base + "/v1/systemone", {**body, **self.options}, timeout=timeout)

    def chat(self, schema, state, timeout=300):
        payload = {"model": "dgemma", "messages": [{"role": "system", "content": json.dumps({**schema, **self.options})},
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
        return {"name": self.name, "kind": self.kind, "role": self.role,
                **({"options": self.options} if self.options else {})}


COMPETITORS = os.path.join(os.path.dirname(os.path.abspath(__file__)), "..", "..", "benchmarks", "competitors")


def load_profile(name):
    path = os.path.join(COMPETITORS, f"{name}.json")
    if not os.path.exists(path):
        raise SystemExit(f"competitor profile {name!r} not found ({os.path.relpath(path)})")
    with open(path) as f:
        return json.load(f)


class CompetitorTarget(Target):
    """Another decision model behind /v1/systemone. Options after "#": profile=<name>, model=<id>, max_options=<N>."""
    role = "competitor"

    def __init__(self, name, url):
        url, _, frag = url.partition("#")
        opts = dict(kv.split("=", 1) for kv in frag.split("&") if kv)
        unknown = set(opts) - {"profile", "model", "max_options"}
        if unknown:
            raise SystemExit(f"competitor {name}: unknown option(s) {sorted(unknown)} (known: profile, model, max_options)")
        self.profile_name = opts.get("profile")
        self.profile = load_profile(self.profile_name) if self.profile_name else {}
        serving = self.profile.get("serving", {})
        self.model = opts.get("model") or serving.get("model_field")
        self.max_options = int(opts.get("max_options") or serving.get("max_choice_options") or 255)
        super().__init__(name, url)  # no dgem request options for a competitor
        self.options = {}

    def systemone(self, body, timeout=300):
        b = {k: v for k, v in body.items() if k not in ("samples", "seed", "layout")}
        if self.model:
            b["model"] = self.model
        return net.request(self.base + "/v1/systemone", b, timeout=timeout)

    def chat(self, schema, state, timeout=300):
        raise RuntimeError("competitor targets only speak /v1/systemone")

    def redacted(self):
        return {"name": self.name, "kind": self.kind, "role": self.role,
                **({"profile": self.profile_name} if self.profile_name else {}),
                **({"model": self.model} if self.model else {})}
