import os
import vllm

vllm_path = os.path.dirname(vllm.__file__)

# 1. Patch vllm/config/vllm.py to register DiffusionAsyncScheduler
vllm_config_file = os.path.join(vllm_path, "config", "vllm.py")
if os.path.exists(vllm_config_file):
    with open(vllm_config_file, "r") as f:
        config_content = f.read()

    scheduler_hook = """
        if (
            self.diffusion_config is not None
            and self.scheduler_config.async_scheduling
            and self.scheduler_config.scheduler_cls is None
        ):
            self.scheduler_config.scheduler_cls = (
                "vllm.v1.core.sched.diffusion_scheduler.DiffusionAsyncScheduler"
            )
"""
    target_line = "current_platform.check_and_update_config(self)"
    if "DiffusionAsyncScheduler" not in config_content:
        if target_line in config_content:
            config_content = config_content.replace(target_line, target_line + "\n" + scheduler_hook)
            with open(vllm_config_file, "w") as f:
                f.write(config_content)
            print("Patched vllm/config/vllm.py with DiffusionAsyncScheduler hook.")
        else:
            print("Warning: target_line not found in vllm/config/vllm.py")

# 2. Patch vllm/_custom_ops.py to pass empty perm tensor to gptq_marlin_repack
custom_ops_file = os.path.join(vllm_path, "_custom_ops.py")
if os.path.exists(custom_ops_file):
    with open(custom_ops_file, "r") as f:
        ops_content = f.read()

    old_repack = "return torch.ops._C.gptq_marlin_repack(\n        b_q_weight, size_k, size_n, num_bits, is_a_8bit\n    )"
    new_repack = "perm = torch.empty(0, dtype=torch.int, device=b_q_weight.device)\n    return torch.ops._C.gptq_marlin_repack(\n        b_q_weight, perm, size_k, size_n, num_bits, is_a_8bit\n    )"

    if old_repack in ops_content:
        ops_content = ops_content.replace(old_repack, new_repack)
        with open(custom_ops_file, "w") as f:
            f.write(ops_content)
        print("Patched vllm/_custom_ops.py for gptq_marlin_repack.")
    else:
        print("Note: old_repack pattern not found in _custom_ops.py (may already be patched).")

print("vLLM post-overlay patches complete.")
