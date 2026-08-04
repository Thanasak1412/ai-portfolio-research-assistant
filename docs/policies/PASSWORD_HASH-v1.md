# PASSWORD_HASH-v1

**Status:** Approved | **Effective:** 2026-08-04 | **Policy version:** v1

## Selected policy

Passwords use Argon2id with 65,536 KiB memory, 3 iterations, parallelism 2, a cryptographically random 16-byte salt, and a 32-byte derived key. The encoded value uses PHC Argon2id format, version 19, and records memory, iterations, parallelism, salt, and hash. Password input is accepted only from 12 through 1,024 UTF-8 bytes; the minimum is 12 user-visible characters and no composition rule applies.

Verification parses only the expected Argon2id PHC shape. Malformed, unsupported, or over-limit hashes are generic authentication failures. A successful login rehashes when algorithm/version/parameters differ from this policy. If the rehash write fails, the successful login remains valid, a secret-free operational error is emitted, and the next successful login retries the upgrade.

## Benchmark record

Environment: macOS Darwin 25.5.0, arm64, 8 logical CPUs, 16 GiB RAM, Go 1.26.5 host runtime, no container; nine sequential samples per candidate and one four-way selected-candidate run. Inputs were synthetic benchmark bytes and were not credentials.

| Candidate | Median | P95 | Theoretical memory |
|---|---:|---:|---:|
| 32 MiB, 3 iterations, 1 lane | 70.1 ms | 103.0 ms | 32 MiB/request |
| 64 MiB, 3 iterations, 1 lane | 146.3 ms | 156.2 ms | 64 MiB/request |
| 64 MiB, 3 iterations, 2 lanes | 78.7 ms | 83.2 ms | 64 MiB/request |
| 96 MiB, 3 iterations, 2 lanes | 123.2 ms | 131.8 ms | 96 MiB/request |

The selected candidate completed four concurrent requests in 183.0 ms wall time with a theoretical 256 MiB concurrent working set. It balances memory hardness with sustainable burst capacity on the measured API-equivalent host. Expected sustained throughput is deployment-dependent; production capacity testing is required before increasing auth concurrency limits.

Rejected alternatives: 32 MiB has lower memory cost; 96 MiB materially reduces burst headroom without a proportionate local security benefit. Parameters must be rebenchmarked on every production architecture change, Go/Argon2 upgrade, a 2x capacity change, or a material authentication DoS finding.

## Monitoring and tests

Measure hash/verify latency histograms and rehash counts without credential labels. Test minimum/maximum input, malformed hash, wrong password, parameter mismatch rehash, rehash persistence failure, and absence of password/hash data in logs or audits.
