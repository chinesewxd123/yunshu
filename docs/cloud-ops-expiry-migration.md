# Cloud Ops & Expiry Migration

## Scope

- Cloud provider dictionaries for Alibaba/Tencent/JD AK/SK and cloud server credential templates.
- Cloud account/server form dictionary label echo fields.
- Cloud server action API: reset password and reboot.
- Cloud expiry rule model and evaluator pipeline.

## Migration steps

1. Pull latest code and restart backend once to trigger schema auto-migration.
2. Run `go run ./cmd/seed.go` to seed new permissions and dictionary presets.
3. Grant new permissions to target roles:
   - `/api/v1/projects/:id/servers/:serverId/cloud-actions` (`POST`)
   - `/api/v1/alerts/cloud-expiry-rules` (`GET`/`POST`)
   - `/api/v1/alerts/cloud-expiry-rules/:id` (`PUT`/`DELETE`)
4. In data dictionary, replace demo AK/SK/password values with real values in production.

## Verification checklist

- Cloud account editor supports dict-fill AK/SK and label echo.
- Cloud server editor supports vendor-specific dict-fill for username/password/private-key/port.
- Cloud server list action buttons can invoke reboot and password reset successfully.
- Creating a cloud expiry rule triggers alert events when instance expiry is within threshold.
- Resolving condition (days left > threshold) emits resolved notifications.

