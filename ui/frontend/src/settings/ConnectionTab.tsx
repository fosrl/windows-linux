import { useState } from "react";
import { SettingsService, type Settings, type SettingsResult } from "@bindings";
import { Alert, Button, Form, LinkRow, Row, Section, Sheet, Switch, TextField, Value } from "../components/controls";
import { report, urls } from "../lib";
import { AdministratorNotice } from "./GeneralTab";
import { BannerRow, BannerText, BuildingIcon, GlobeIcon, LockIcon, SettingTitle } from "./banner";

const defaultMTU = "1280";

type EditField = "primaryDns" | "secondaryDns" | "mtu";

/** The settings on this tab that the org's server can manage, by JSON name. */
const managedFields: (keyof Settings)[] = [
  "dnsOverride",
  "dnsTunnel",
  "primaryDns",
  "secondaryDns",
  "mtu",
  "exitNodeTakesPrecedence",
];

/**
 * How many of this tab's settings the org's server manages. The two upstream
 * DNS servers count as one, since they're managed together.
 */
function managedCount(managed: string[]) {
  const counted = managed
    .filter((field) => managedFields.includes(field as keyof Settings))
    .map((field) => (field === "secondaryDns" ? "primaryDns" : field));
  return new Set(counted).size;
}

/**
 * The tunnel settings, which can differ per org. Changes are saved for all
 * organizations until the current org gets its own settings with the banner's
 * button; after that they're saved only for it. Settings apply as soon as they
 * change, like the macOS app.
 */
export function ConnectionTab({
  settings,
  apply,
  onChange,
}: {
  settings: Settings;
  apply: (patch: Partial<Settings>) => Promise<SettingsResult>;
  /** Replaces the shown settings after switching between all organizations and the org's own. */
  onChange: (settings: Settings) => void;
}) {
  const [editing, setEditing] = useState<EditField | null>(null);
  const [confirmingReset, setConfirmingReset] = useState(false);
  const disabled = settings.disabled;
  const managed = settings.managed ?? [];
  const orgName = settings.orgName;

  /** Whether the org's server sets a setting for the user's role. */
  const managedByServer = (field: keyof Settings) => managed.includes(field);
  /** Whether a setting is read-only, because of an administrator or the org's server. */
  const locked = (field: keyof Settings) => disabled || managedByServer(field);
  /** A setting's name, with a lock when the org's server manages it. */
  const title = (text: string, field: keyof Settings) => (
    <SettingTitle locked={managedByServer(field)} orgName={orgName}>
      {text}
    </SettingTitle>
  );

  const toggle = (patch: Partial<Settings>) => report(apply(patch));
  const useOwnSettings = () => report(SettingsService.UseOwnSettings().then(onChange));
  const resetOrg = () => {
    setConfirmingReset(false);
    report(SettingsService.ResetOrg().then(onChange));
  };
  const count = managedCount(managed);

  return (
    <>
      <Form>
        {disabled && <AdministratorNotice />}

        {orgName && (
          <Section>
            {settings.orgMode ? (
              <BannerRow
                action={
                  <Button size="small" disabled={disabled} onClick={() => setConfirmingReset(true)}>
                    Reset…
                  </Button>
                }
              >
                <BannerText
                  icon={<BuildingIcon />}
                  title={`${orgName} has its own settings`}
                  detail={`Changes are saved only for ${orgName}. Other organizations use your settings for all organizations.`}
                />
              </BannerRow>
            ) : (
              <BannerRow
                action={
                  <Button size="small" disabled={disabled} onClick={useOwnSettings}>
                    Use Own Settings
                  </Button>
                }
              >
                <BannerText
                  icon={<GlobeIcon />}
                  title={`${orgName} uses your settings for all organizations`}
                  detail={`Changes apply to every organization. Own settings for ${orgName} start as a copy of these.`}
                />
              </BannerRow>
            )}
            {count > 0 && (
              <BannerRow>
                <BannerText
                  icon={<LockIcon />}
                  title={`Managed by your administrator in ${orgName}`}
                  detail={`Your administrator set ${count === 1 ? "1 of these settings" : `${count} of these settings`} for your role. To change ${count === 1 ? "it" : "them"}, contact your administrator.`}
                />
              </BannerRow>
            )}
          </Section>
        )}

        <Section header="DNS Settings">
          <Row
            title={title("Enable Aliases (DNS Override)", "dnsOverride")}
            description="When enabled, the client uses custom DNS servers to resolve internal resources and aliases. This overrides your system’s default DNS settings. Queries that cannot be resolved as a Pangolin resource will be forwarded to your configured Upstream DNS Server."
          >
            <Switch
              label="Enable Aliases (DNS Override)"
              checked={settings.dnsOverride}
              disabled={locked("dnsOverride")}
              onChange={(v) => toggle({ dnsOverride: v })}
            />
          </Row>
          <Row
            title={title("DNS Over Tunnel", "dnsTunnel")}
            description="When enabled, DNS queries are routed through the tunnel for remote resolution. To ensure queries are tunneled correctly, you must define the DNS server as a Pangolin resource and enter its address as an Upstream DNS Server."
          >
            <Switch
              label="DNS Over Tunnel"
              checked={settings.dnsTunnel}
              disabled={locked("dnsTunnel")}
              onChange={(v) => toggle({ dnsTunnel: v })}
            />
          </Row>
          <Row title={title("Primary Upstream DNS Server", "primaryDns")}>
            <Value>{settings.primaryDns || "System DNS"}</Value>
            <Button size="small" disabled={locked("primaryDns")} onClick={() => setEditing("primaryDns")}>
              Set...
            </Button>
          </Row>
          <Row title={title("Secondary Upstream DNS Server", "secondaryDns")}>
            <Value>{settings.secondaryDns || "System DNS"}</Value>
            <Button size="small" disabled={locked("secondaryDns")} onClick={() => setEditing("secondaryDns")}>
              Set...
            </Button>
          </Row>
        </Section>

        <Section header="Advanced">
          <Row
            title={title("Exit Node Takes Precedence Over Resources", "exitNodeTakesPrecedence")}
            description="When enabled, routes for individual resources are not added to the system and their aliases are not resolved, so all traffic is sent through the exit node instead of directly to resources."
          >
            <Switch
              label="Exit Node Takes Precedence Over Resources"
              checked={settings.exitNodeTakesPrecedence}
              disabled={locked("exitNodeTakesPrecedence")}
              onChange={(v) => toggle({ exitNodeTakesPrecedence: v })}
            />
          </Row>
          <Row title={title("MTU", "mtu")} description="Your sites must be configured to use the same MTU value.">
            <Value>{settings.mtu}</Value>
            <Button size="small" disabled={locked("mtu")} onClick={() => setEditing("mtu")}>
              Set...
            </Button>
          </Row>
        </Section>

        <Section header="Help">
          <LinkRow href={urls.configureClient}>See docs for more info on these settings</LinkRow>
        </Section>
      </Form>

      {editing === "mtu" && (
        <ValueSheet
          title="MTU:"
          initial={settings.mtu}
          hint="Enter an integer between 576 and 9000 (e.g., 1280)"
          resetLabel="Default"
          resetValue={defaultMTU}
          onClose={() => setEditing(null)}
          onSave={(v) => apply({ mtu: v })}
          field="mtu"
        />
      )}
      {(editing === "primaryDns" || editing === "secondaryDns") && (
        <ValueSheet
          title={editing === "primaryDns" ? "Primary Upstream DNS Server:" : "Secondary Upstream DNS Server:"}
          initial={settings[editing]}
          placeholder="System DNS"
          resetLabel="Use System DNS"
          resetValue=""
          resetSaves
          onClose={() => setEditing(null)}
          onSave={(v) => apply({ [editing]: v })}
          field={editing}
        />
      )}
      {confirmingReset && (
        <Alert
          title={`Reset ${orgName} to settings for all organizations?`}
          message={`${orgName} will use your settings for all organizations again. Settings you changed for ${orgName} will be removed.`}
          confirmLabel="Reset"
          destructive
          onConfirm={resetOrg}
          onCancel={() => setConfirmingReset(false)}
        />
      )}
    </>
  );
}

/** The DNS and MTU editing sheets from the macOS app. */
function ValueSheet({
  title,
  initial,
  placeholder,
  hint,
  resetLabel,
  resetValue,
  resetSaves,
  field,
  onSave,
  onClose,
}: {
  title: string;
  initial: string;
  placeholder?: string;
  hint?: string;
  resetLabel: string;
  resetValue: string;
  /** "Use System DNS" saves right away; "Default" only fills the field. */
  resetSaves?: boolean;
  field: EditField;
  onSave: (value: string) => Promise<SettingsResult>;
  onClose: () => void;
}) {
  const [value, setValue] = useState(initial);
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);

  const save = async (v: string) => {
    setBusy(true);
    try {
      const result = await onSave(v.trim());
      if (result.field === field && result.error) setError(result.error);
      else onClose();
    } catch (err) {
      console.error(err);
    } finally {
      setBusy(false);
    }
  };

  return (
    <Sheet
      onCancel={onClose}
      onSubmit={() => save(value)}
      footer={
        <>
          <Button
            onClick={() => {
              setValue(resetValue);
              setError("");
              if (resetSaves) void save(resetValue);
            }}
          >
            {resetLabel}
          </Button>
          <div className="flex-1" />
          <Button onClick={onClose}>Cancel</Button>
          <Button type="submit" variant="prominent" disabled={busy}>
            Done
          </Button>
        </>
      }
    >
      <label className="text-[13px]" htmlFor="sheet-value">
        {title}
      </label>
      <TextField
        id="sheet-value"
        autoFocus
        value={value}
        placeholder={placeholder}
        onChange={(e) => {
          setValue(e.target.value);
          setError("");
        }}
        className="w-full"
      />
      {(error || hint) && (
        <div className={error ? "text-[11px] text-mac-danger" : "text-[11px] text-mac-secondary"}>{error || hint}</div>
      )}
    </Sheet>
  );
}
