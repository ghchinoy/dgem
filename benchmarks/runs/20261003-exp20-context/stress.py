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

import sys,json,time,concurrent.futures as cf
sys.argv_saved=sys.argv; sys.argv=["x"]; exec(open("long_slice.py").read().split("if __name__")[0]); sys.argv=sys.argv_saved
from matrix.targets import Target
t=Target("ctx32k",sys.argv[1]); cs=[padded(c,"p20k") for c in base_cases()]
for w in (8,16):
    jobs=(cs*2)[:max(64,w*4)]; t0=time.time()
    with cf.ThreadPoolExecutor(w) as ex: res=list(ex.map(lambda c:t.systemone(C.body(c),timeout=900)[:3:2],jobs))
    ok=sorted(ms for st,ms in res if st==200)
    print(json.dumps({"w":w,"n":len(res),"ok":len(ok),"errors":sorted({str(st) for st,_ in res if st!=200}),"p50":round(ok[len(ok)//2]) if ok else None,"p90":round(ok[int(.9*len(ok))]) if ok else None,"wall_s":round(time.time()-t0,1)}))
