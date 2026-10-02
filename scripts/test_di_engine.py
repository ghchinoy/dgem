#!/usr/bin/env python3
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

"""Smoke test verifying dgem systemone adapter compatibility with decision_index.engines.http:HttpSystemOne."""

import http.server
import json
import socketserver
import subprocess
import sys
import threading
import time
import urllib.request

VLLM_PORT = 8097
ADAPTER_PORT = 8098

class MockVLLMHandler(http.server.BaseHTTPRequestHandler):
    def do_POST(self):
        length = int(self.headers.get("Content-Length", 0))
        body = json.loads(self.rfile.read(length))
        
        # Check system message
        sys_content = body["messages"][0]["content"]
        schema = json.loads(sys_content)
        
        answers = {}
        for q in schema.get("questions", []):
            qid = q["id"]
            qtype = q.get("type", "choice")
            if qtype in ("boolean", "noul", "bool"):
                answers[qid] = {
                    "type": "boolean",
                    "label": "yes",
                    "confidence": 0.92,
                    "noul": 0.92,
                    "probabilities": {"yes": 0.92, "no": 0.08}
                }
            else:
                opts = [o["name"] for o in q.get("options", [])]
                chosen = opts[0]
                probs = {o: 0.10 / max(1, len(opts)-1) for o in opts}
                probs[chosen] = 0.90
                answers[qid] = {
                    "type": "choice",
                    "label": chosen,
                    "choice": chosen,
                    "confidence": 0.90,
                    "probabilities": probs
                }
        
        resp_obj = {
            "choices": [
                {
                    "message": {
                        "role": "assistant",
                        "content": json.dumps({"answers": answers})
                    }
                }
            ]
        }
        resp_bytes = json.dumps(resp_obj).encode("utf-8")
        self.send_response(200)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(resp_bytes)))
        self.end_headers()
        self.wfile.write(resp_bytes)

    def log_message(self, format, *args):
        pass

def main():
    print("==> Starting mock vLLM server on port", VLLM_PORT)
    vllm_srv = socketserver.TCPServer(("127.0.0.1", VLLM_PORT), MockVLLMHandler)
    vllm_thread = threading.Thread(target=vllm_srv.serve_forever, daemon=True)
    vllm_thread.start()

    print("==> Starting dgem systemone serve on port", ADAPTER_PORT)
    adapter_proc = subprocess.Popen([
        "./bin/dgem", "systemone", "serve",
        "--port", str(ADAPTER_PORT),
        "--upstream", f"http://127.0.0.1:{VLLM_PORT}/v1",
        "--temperature", "1.0"
    ], stdout=subprocess.PIPE, stderr=subprocess.PIPE)

    try:
        # Wait for adapter to become ready
        ready = False
        for _ in range(20):
            try:
                with urllib.request.urlopen(f"http://127.0.0.1:{ADAPTER_PORT}/health", timeout=1) as resp:
                    if resp.status == 200:
                        ready = True
                        break
            except Exception:
                time.sleep(0.2)
        
        if not ready:
            print("ERROR: dgem systemone adapter failed to become ready!")
            sys.exit(1)
        print("==> dgem systemone adapter is healthy!")

        # Import HttpSystemOne from di-work
        sys.path.insert(0, "/Users/ghchinoy/di-work/decision-index")
        from decision_index.engines.http import HttpSystemOne
        from decision_index.engines.base import validate

        engine = HttpSystemOne(base_url=f"http://127.0.0.1:{ADAPTER_PORT}", model="dgem")

        # Test Case 1: Standard Choice Question
        state = "Customer was double billed on transaction #48291."
        q_choice = {
            "department": {
                "type": "choice",
                "instructions": "Route to the responsible department",
                "criteria": {
                    "billing": "Invoice and charge disputes",
                    "engineering": "Platform and bug reports",
                    "support": "General questions"
                }
            }
        }
        res_choice, _ = engine(state, q_choice)
        validate(q_choice, res_choice)
        print("✓ Choice validation PASSED:", res_choice["answers"]["department"]["choice"])

        # Test Case 2: Noul Question (RAGTruth / PhishNChips format)
        q_noul = {
            "is_hallucinated": {
                "type": "noul",
                "instructions": "Is this statement supported by context?"
            }
        }
        res_noul, _ = engine(state, q_noul)
        validate(q_noul, res_noul)
        print("✓ Noul validation PASSED: p_yes =", res_noul["answers"]["is_hallucinated"]["noul"])

        # Test Case 3: Mixed Choice + Noul in one request
        q_mixed = {
            "urgent": {"type": "noul", "instructions": "Is this urgent?"},
            "category": {
                "type": "choice",
                "instructions": "Select category",
                "criteria": {"dispute": "Billing dispute", "inquiry": "General inquiry"}
            }
        }
        res_mixed, _ = engine(state, q_mixed)
        validate(q_mixed, res_mixed)
        print("✓ Mixed Choice+Noul validation PASSED!")

        print("\n🎉 ALL DECISION INDEX ENGINE VALIDATION CHECKS PASSED!")
    finally:
        adapter_proc.terminate()
        vllm_srv.shutdown()

if __name__ == "__main__":
    main()
