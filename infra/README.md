# infra

**OpenTofu IaC** for cloud resources (v1.12.6, `tofu`): GCP Cloud Run,
Artifact Registry, Cloud SQL for the testing/staging platform, scaled to zero
when idle (`min_instance_count = 0`). Edge / CDN configuration for Cloudflare
or bunny.net (TLS, proxying, caching) will be managed alongside deployment.
Contents land with the deploy epic; nothing is provisioned from here during Story 1.1.
