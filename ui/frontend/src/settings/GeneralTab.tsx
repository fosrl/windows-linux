import type { Settings, SettingsResult } from "@bindings";
import { Form, Row, Section, Switch } from "../components/controls";
import { isLinux, report } from "../lib";

/**
 * App-wide settings. Unlike the Connection tab's, these are the same for every
 * organization and account. Settings apply as soon as they change, like the
 * macOS app.
 */
export function GeneralTab({
  settings,
  apply,
}: {
  settings: Settings;
  apply: (patch: Partial<Settings>) => Promise<SettingsResult>;
}) {
  const disabled = settings.disabled;
  const toggle = (patch: Partial<Settings>) => report(apply(patch));

  return (
    <Form>
      {disabled && <AdministratorNotice />}

      <Section header="Startup">
        <Row title="Start at Login" description={`Starts Pangolin when you sign in${isLinux ? "" : " to Windows"}.`}>
          <Switch
            label="Start at Login"
            checked={settings.openAtLogin}
            disabled={disabled}
            // Turning off start at login also turns off connect at start.
            onChange={(v) => toggle({ openAtLogin: v, autoConnect: v ? settings.autoConnect : false })}
          />
        </Row>
        <Row
          title="Connect at Start"
          description="Connects the tunnel whenever Pangolin starts. Also opens Pangolin at sign-in."
        >
          <Switch
            label="Connect at Start"
            checked={settings.autoConnect}
            disabled={disabled}
            // Connect at start implies start at login.
            onChange={(v) => toggle({ autoConnect: v, openAtLogin: v ? true : settings.openAtLogin })}
          />
        </Row>
      </Section>

      <Section header="Updates">
        <Row
          title="Automatically Download and Install Updates"
          description={
            settings.autoDownloadUpdatesUnavailable ||
            "Downloads updates in the background and installs them the next time Pangolin quits or starts."
          }
        >
          <Switch
            label="Automatically Download and Install Updates"
            checked={settings.autoDownloadUpdates && !settings.autoDownloadUpdatesUnavailable}
            disabled={disabled || !!settings.autoDownloadUpdatesUnavailable}
            onChange={(v) => toggle({ autoDownloadUpdates: v })}
          />
        </Row>
      </Section>
    </Form>
  );
}

/** Shown on both tabs when an administrator has turned off user settings. */
export function AdministratorNotice() {
  return (
    <div className="rounded-[10px] border border-mac-group-border bg-mac-group px-3 py-2 text-[12px] text-mac-secondary">
      These settings are managed by your administrator.
    </div>
  );
}
