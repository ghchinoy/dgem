import os
import sys
import vllm

vllm_dir = os.path.dirname(vllm.__file__)
custom_ops_path = os.path.join(vllm_dir, "_custom_ops.py")

if not os.path.exists(custom_ops_path):
    print(f"Error: {custom_ops_path} does not exist!")
    sys.exit(1)

with open(custom_ops_path, "r") as f:
    code = f.read()

target = "    return torch.ops._C.gptq_marlin_repack(\n        b_q_weight, size_k, size_n, num_bits, is_a_8bit\n    )"
replacement = "    perm = torch.empty(0, dtype=torch.int, device=b_q_weight.device)\n    return torch.ops._C.gptq_marlin_repack(\n        b_q_weight, perm, size_k, size_n, num_bits, is_a_8bit\n    )"

if target in code:
    code = code.replace(target, replacement)
    with open(custom_ops_path, "w") as f:
        f.write(code)
    print("Successfully patched _custom_ops.py for gptq_marlin_repack!")
else:
    print("Error: Target code snippet not found in _custom_ops.py!")
    sys.exit(1)
