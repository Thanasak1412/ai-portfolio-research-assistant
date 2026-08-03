# Release and Rollback Procedure

1. Require all protected-branch quality gates and review.
2. Build immutable API, worker, and web artifacts from one revision.
3. Back up the production database according to the environment's recovery policy.
4. Apply reviewed forward migrations before enabling code that needs them.
5. Deploy API/worker/web from the same release revision and verify liveness/readiness.
6. Roll back application artifacts when runtime verification fails. Database rollback is performed only when the owning migration documents it as safe; otherwise use an approved forward repair.

M0 defines the procedure but does not publish or deploy an environment. Production backup retention, RPO, and RTO remain an operations-owner decision before production launch.

