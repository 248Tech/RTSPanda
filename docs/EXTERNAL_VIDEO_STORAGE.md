# External Video Storage

RTSPanda can auto-sync recorded `.mp4` files from `DATA_DIR/recordings` to:

- Local Server (NAS/SMB/NFS mounted path)
- Dropbox
- Google Drive
- OneDrive
- Proton Drive

Configuration is in **Settings -> Integrations -> External Video Storage**.

## How it works

- RTSPanda records locally first.
- A background worker copies only completed files (older than your configured minimum file age).
- Sync runs on your configured interval.
- Local Server mode copies files directly to a filesystem path.
- Cloud providers use `rclone copy`.

## Common prerequisites

1. Install `rclone` on the RTSPanda host machine.
2. Ensure `rclone` is reachable in `PATH`, or set `RCLONE_BIN` to its full path.
3. In RTSPanda, open **Settings -> Integrations -> External Video Storage**.
4. Enable sync and set:
   - `Sync interval (seconds)` (30-86400)
   - `Min file age before upload (seconds)` (15-3600)

## Solution 1: Local Server (NAS/SMB/NFS)

Use this when your server exposes a folder path mounted on the RTSPanda host.

1. Choose provider: `Local Server (NAS/SMB/NFS)`.
2. Set `Local server destination path`, for example:
   - Windows UNC path: `\\nas\camera-archive\rtspanda`
   - Linux mount path: `/mnt/nas/camera-archive/rtspanda`
3. Save integrations.

RTSPanda will mirror folder structure like `camera-{id}/YYYY-MM-DD_HH-MM-SS.mp4`.

## Solution 2: Dropbox

1. Create an rclone remote (interactive):
   - `rclone config`
   - New remote name: `dropbox`
   - Storage type: `Dropbox`
2. Verify remote:
   - `rclone lsd dropbox:`
3. In RTSPanda:
   - Provider: `Dropbox (rclone)`
   - rclone remote name: `dropbox`
   - Remote folder path: e.g. `RTSPanda`
4. Save integrations.

## Solution 3: Google Drive

1. Create an rclone remote:
   - `rclone config`
   - New remote name: `gdrive`
   - Storage type: `Google Drive`
2. Verify:
   - `rclone lsd gdrive:`
3. In RTSPanda:
   - Provider: `Google Drive (rclone)`
   - rclone remote name: `gdrive`
   - Remote folder path: e.g. `RTSPanda`
4. Save integrations.

## Solution 4: OneDrive

1. Create an rclone remote:
   - `rclone config`
   - New remote name: `onedrive`
   - Storage type: `OneDrive`
2. Verify:
   - `rclone lsd onedrive:`
3. In RTSPanda:
   - Provider: `OneDrive (rclone)`
   - rclone remote name: `onedrive`
   - Remote folder path: e.g. `RTSPanda`
4. Save integrations.

## Solution 5: Proton Drive

1. Use a recent rclone build with Proton Drive backend support.
2. Create an rclone remote:
   - `rclone config`
   - New remote name: `protondrive`
   - Storage type: `Proton Drive`
3. Verify:
   - `rclone lsd protondrive:`
4. In RTSPanda:
   - Provider: `Proton Drive (rclone)`
   - rclone remote name: `protondrive`
   - Remote folder path: e.g. `RTSPanda`
5. Save integrations.

## Quick validation

1. Enable recording on a camera and wait for segments to appear in `DATA_DIR/recordings`.
2. Confirm RTSPanda logs include `video-storage: sync complete`.
3. Verify files appear at your configured destination.

## Troubleshooting

- `rclone copy failed`:
  - Validate remote with `rclone lsd <remote>:`.
  - Re-run `rclone config` if token/auth expired.
- Nothing uploads:
  - Confirm external storage is enabled in Integrations.
  - Check min file age is not larger than expected.
  - Check sync interval and backend logs.
- Local server errors:
  - Ensure destination path is writable by the RTSPanda process user.
