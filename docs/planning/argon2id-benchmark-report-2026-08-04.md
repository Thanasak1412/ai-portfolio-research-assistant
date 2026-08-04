# Argon2id Benchmark Report — 2026-08-04

**Status:** Approved supporting evidence
**Policy reference:** [PASSWORD_HASH-v1](../policies/PASSWORD_HASH-v1.md) and ADR-014

## Method

This decision benchmark used Go 1.26.5 on Darwin 25.5.0 arm64 (8 logical CPUs, 16 GiB RAM), executed on the host as the closest reproducible API-runtime equivalent available before M1. Each candidate had nine sequential samples. The selected candidate also had one four-request concurrent run. Inputs were synthetic byte buffers, never user credentials. Results measure wall-clock hashing time; they are not a production capacity certification.

| Memory | Iterations | Lanes | Median | P95 | Working memory/request |
|---|---:|---:|---:|---:|---:|
| 32 MiB | 3 | 1 | 70.1 ms | 103.0 ms | 32 MiB |
| 64 MiB | 3 | 1 | 146.3 ms | 156.2 ms | 64 MiB |
| 64 MiB | 3 | 2 | 78.7 ms | 83.2 ms | 64 MiB |
| 96 MiB | 3 | 2 | 123.2 ms | 131.8 ms | 96 MiB |

The 64 MiB, three-iteration, two-lane candidate completed a four-request run in 183.0 ms wall time. Its theoretical concurrent memory demand is 256 MiB.

## Decision

Select 64 MiB memory, three iterations, two lanes, a 16-byte random salt, and a 32-byte derived key. The candidate provides memory-hard verification while retaining practical burst headroom on the measured machine. Thirty-two MiB was rejected for lower memory cost; 96 MiB was rejected because it reduced burst headroom without a proportionate demonstrated benefit.

Before production release, repeat the benchmark on the target API instance class with expected concurrent login traffic, and verify memory limits and latency SLOs. Rebenchmark on production architecture or Go/Argon2 changes, a twofold authentication-capacity change, or a material authentication DoS finding.
