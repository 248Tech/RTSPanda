const sectionClass =
  'rounded-xl border border-border bg-card p-4 sm:p-5'

export default function Guides() {
  return (
    <div className="space-y-6">
      <header className="rounded-xl border border-accent/30 bg-accent/10 p-5">
        <div className="flex flex-wrap items-center justify-between gap-3">
          <div>
            <h1 className="text-xl font-semibold text-text-primary">Guides</h1>
            <p className="mt-1 text-sm text-text-muted">
              Practical setup docs for Lorex, secure remote access, and Frigate alert integration.
            </p>
          </div>
          <a
            href="https://248tech.com/donate"
            target="_blank"
            rel="noopener noreferrer"
            className="inline-flex items-center gap-2 rounded-lg bg-accent px-4 py-2 text-sm font-semibold text-white transition-opacity hover:opacity-90 focus:outline-none focus:ring-2 focus:ring-accent"
          >
            Support the Developer
          </a>
        </div>
      </header>

      <section className={sectionClass}>
        <h2 className="text-lg font-semibold text-text-primary">
          Major Update: Frigate Alerts (YOLO or Frigate)
        </h2>
        <p className="mt-2 text-sm text-text-muted">
          RTSPanda now supports provider-based detection alerts. When editing a camera, open
          <span className="mx-1 rounded bg-base px-1 font-mono text-xs">Discord Rich Alerts</span>
          and choose:
        </p>
        <ul className="mt-3 list-disc space-y-1 pl-5 text-sm text-text-primary">
          <li><strong>RTSPanda YOLOv8</strong>: built-in detector and overlays.</li>
          <li><strong>Frigate</strong>: external Frigate detections sent to RTSPanda webhook.</li>
        </ul>
        <div className="mt-3 rounded-lg border border-border bg-base p-3 text-sm text-text-muted">
          <p className="font-medium text-text-primary">Frigate webhook endpoint</p>
          <p className="mt-1">
            <code className="rounded bg-card px-1 font-mono text-xs text-accent">
              POST /api/v1/frigate/events
            </code>
          </p>
          <p className="mt-2">
            Optional snapshot attach: set <code className="rounded bg-card px-1 font-mono text-xs text-accent">FRIGATE_BASE_URL</code>{' '}
            so RTSPanda can fetch event snapshots from Frigate.
          </p>
        </div>
      </section>

      <section className={sectionClass}>
        <h2 className="text-lg font-semibold text-text-primary">
          Guide 1: Port Forward a Lorex NVR and Add It to RTSPanda
        </h2>
        <ol className="mt-3 list-decimal space-y-2 pl-5 text-sm text-text-primary">
          <li>Give your Lorex NVR a static LAN IP address (for example, <code className="rounded bg-base px-1 font-mono text-xs">192.168.1.200</code>).</li>
          <li>Log into your router and create NAT/port-forward rules to that NVR IP for Lorex ports you use (commonly RTSP <code className="rounded bg-base px-1 font-mono text-xs">554</code> and your NVR web/admin port).</li>
          <li>Set a strong NVR password and disable default accounts before exposing any port.</li>
          <li>Find your public IP or DDNS hostname, then test from an external network.</li>
          <li>Add the stream in RTSPanda as: <code className="rounded bg-base px-1 font-mono text-xs">rtsp://user:pass@YOUR_PUBLIC_IP:554/cam/realmonitor?channel=1&subtype=0</code></li>
        </ol>
        <p className="mt-3 rounded-lg border border-status-offline/30 bg-status-offline/10 p-3 text-sm text-text-primary">
          Direct port forwarding is high risk. Prefer Tailscale (below) for secure remote access.
        </p>
      </section>

      <section className={sectionClass}>
        <h2 className="text-lg font-semibold text-text-primary">
          Guide 2: Tailscale Setup (Recommended Remote Access)
        </h2>
        <ol className="mt-3 list-decimal space-y-2 pl-5 text-sm text-text-primary">
          <li>Install Tailscale on the RTSPanda host and on your remote phone/laptop.</li>
          <li>Sign in to both devices with the same tailnet account.</li>
          <li>On the RTSPanda host, verify Tailscale IP with <code className="rounded bg-base px-1 font-mono text-xs">tailscale ip -4</code>.</li>
          <li>Access RTSPanda remotely using <code className="rounded bg-base px-1 font-mono text-xs">http://TAILSCALE_IP:8080</code>.</li>
          <li>If needed, allow <code className="rounded bg-base px-1 font-mono text-xs">8080/tcp</code> in host firewall for the Tailscale interface.</li>
        </ol>
      </section>

      <section className={sectionClass}>
        <h2 className="text-lg font-semibold text-text-primary">
          Guide 3: Get Your RTSP Address from Lorex Camera Login
        </h2>
        <ol className="mt-3 list-decimal space-y-2 pl-5 text-sm text-text-primary">
          <li>Log into Lorex NVR web UI (or desktop client) with an admin account.</li>
          <li>Open camera/network settings and ensure RTSP is enabled.</li>
          <li>Get camera channel number and stream profile (main/sub stream).</li>
          <li>Build URL format (common Lorex/Dahua style): <code className="rounded bg-base px-1 font-mono text-xs">rtsp://USER:PASS@NVR_IP:554/cam/realmonitor?channel=1&subtype=0</code></li>
          <li>Test URL in VLC first, then paste it into RTSPanda camera setup.</li>
        </ol>
      </section>
    </div>
  )
}
