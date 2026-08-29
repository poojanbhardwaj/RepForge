# ADR 0005: AWS production reference

Status: reference only; no resources authorized.

Use separate AWS accounts and Terraform state for staging/production, ECS Fargate for API/web/worker, RDS PostgreSQL, managed Redis, S3/CloudFront, WAF/ALB, KMS/Secrets Manager, backups, and private networking in `ap-south-1`. ECS is preferred over Kubernetes for the initial modular monolith and worker. Terraform starts only when a deployable staging milestone is approved.
