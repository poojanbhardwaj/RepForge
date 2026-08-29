# ADR 0004: Garage for local S3 compatibility

Status: accepted for local/test only.

Use Garage single-node mode behind an S3 interface because the MinIO community server is archived and LocalStack requires an external account/token. Garage is bound to loopback and stores synthetic data only. Production uses managed AWS S3; Garage is never a production recommendation.
