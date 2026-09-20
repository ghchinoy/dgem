import pathlib

p = pathlib.Path("/usr/local/lib/python3.12/dist-packages/vllm/v1/worker/gpu_worker.py")
if not p.exists():
    # Check alternate site-packages path
    import importlib.util
    p = pathlib.Path(importlib.util.find_spec("vllm").origin).parent / "v1" / "worker" / "gpu_worker.py"

s = p.read_text()

# 1. Skip profile_run in eager mode
s = s.replace(
    "        if kv_cache_memory_bytes := self.cache_config.kv_cache_memory_bytes:\n"
    "            # still need a profile run which compiles the model for\n"
    "            # max_num_batched_tokens\n"
    "            self.model_runner.profile_run()",
    "        if kv_cache_memory_bytes := self.cache_config.kv_cache_memory_bytes:\n"
    "            if not self.model_config.enforce_eager:\n"
    "                self.model_runner.profile_run()"
)

# 2. Skip kernel_warmup in eager mode
s = s.replace(
    "        kernel_warmup(self)\n\n"
    "        if self.use_v2_model_runner:\n"
    "            # A workspace resize after capture frees what the graphs point at.\n"
    "            warmup_kernels(self.model_runner, self.execute_model, self.sample_tokens)",
    "        if not self.model_config.enforce_eager:\n"
    "            kernel_warmup(self)\n"
    "            if self.use_v2_model_runner:\n"
    "                warmup_kernels(self.model_runner, self.execute_model, self.sample_tokens)"
)

# 3. Synchronize with background /dev/shm copy ONLY if model path starts with /dev/shm
s = s.replace(
    "self.model_runner.load_model(load_dummy_weights=load_dummy_weights)",
    "import os, time\n"
    "            if os.environ.get('MODEL', '').startswith('/dev/shm'):\n"
    "                while not os.path.exists('/dev/shm/dgemma/.ready'): time.sleep(0.05)\n"
    "            self.model_runner.load_model(load_dummy_weights=load_dummy_weights)"
)

p.write_text(s)
print("[patch_vllm] Successfully patched gpu_worker.py for eager Cloud Run execution")
