# Migrations

M0 contains one no-op platform migration so goose can establish and verify its migration chain. It creates no application-owned table; goose may create its own version metadata table. Authentication and financial tables belong to their owning feature milestones.
