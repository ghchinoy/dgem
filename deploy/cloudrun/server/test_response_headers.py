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
"""Every JSON response carries an explicit charset and nosniff, whichever path wrote it (health, 404, 401, 400, 410).
Real HTTP, no model.

  python3 -m unittest deploy/cloudrun/server/test_response_headers.py
"""

import http.client
import os
import sys
import threading
import unittest

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from test_server_key import load_server  # noqa: E402


class ResponseHeadersTest(unittest.TestCase):
    def setUp(self):
        mod = load_server("k")
        srv = mod._Server(("127.0.0.1", 0), mod.Handler)
        threading.Thread(target=srv.serve_forever, daemon=True).start()
        self.addCleanup(srv.shutdown)
        self.conn = http.client.HTTPConnection("127.0.0.1", srv.server_address[1], timeout=5)
        self.addCleanup(self.conn.close)

    def check(self, method, path, status, body=None, key=None):
        headers = {"content-type": "application/json"}
        if key:
            headers["X-DGem-Key"] = key
        self.conn.request(method, path, body=body, headers=headers)
        r = self.conn.getresponse()
        r.read()
        self.assertEqual(r.status, status, path)
        self.assertEqual(r.getheader("content-type"), "application/json; charset=utf-8", path)
        self.assertEqual(r.getheader("x-content-type-options"), "nosniff", path)

    def test_headers_on_every_path(self):
        self.check("GET", "/health", 200)
        self.check("GET", "/nope", 404)
        self.check("POST", "/v1/chat/completions", 401, body=b"{}")
        self.check("POST", "/v1/chat/completions", 400, body=b"not json", key="k")
        self.check("POST", "/v1/raw/chat/completions", 410, body=b"{}", key="k")


if __name__ == "__main__":
    unittest.main()
