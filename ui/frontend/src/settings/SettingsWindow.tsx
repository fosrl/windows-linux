import { useEffect, useRef, useState, type ReactNode } from "react";
import {
  SettingsService,
  type LoginRequest,
  type Settings,
  type SettingsOpened,
  type SettingsResult,
} from "@bindings";
import { cx } from "../components/controls";
import { report, useEvent } from "../lib";
import { GeneralTab } from "./GeneralTab";
import { ConnectionTab } from "./ConnectionTab";
import { AccountsTab } from "./AccountsTab";
import { StatusTab } from "./StatusTab";
import { LogsTab } from "./LogsTab";
import { AboutTab } from "./AboutTab";

// Tab indexes are shared with the backend (settingsTab* in ui/windows.go).
const sections: { name: string; icon: ReactNode }[] = [
  { name: "General", icon: <GearIcon /> },
  { name: "Connection", icon: <NetworkIcon /> },
  { name: "Accounts", icon: <AccountsIcon /> },
  { name: "Status", icon: <StatusIcon /> },
  { name: "About", icon: <InfoIcon /> },
];
const generalTab = 0;
const connectionTab = 1;
const accountsTab = 2;
const statusTab = 3;
const aboutTab = 4;

/** Sidebar + detail layout, like the macOS app's NavigationSplitView. */
export function SettingsWindow() {
  const [tab, setTabState] = useState(0);
  // Logs is a page within About, opened from it like a pushed view on macOS.
  const [showingLogs, setShowingLogs] = useState(false);
  // Changing section, or reopening the window, goes back to the section's first page.
  const setTab = (t: number) => {
    setTabState(t);
    setShowingLogs(false);
  };
  // One copy of the settings for the General and Connection tabs: each change
  // saves the whole form, so neither tab may hold a stale copy of the other's.
  const [settings, setSettings] = useState<Settings | null>(null);
  const settingsRef = useRef(settings);
  settingsRef.current = settings;
  // Bumped each time the window is shown, so sections reset like a freshly opened window.
  const [openCount, setOpenCount] = useState(0);
  const [login, setLogin] = useState<LoginRequest | null>(null);
  // Tabs can put a toolbar control at the trailing end of the header.
  const [headerAccessory, setHeaderAccessory] = useState<HTMLElement | null>(null);

  /** Saves a change. Resolves to the backend's result so sheets can show validation errors. */
  const apply = async (patch: Partial<Settings>): Promise<SettingsResult> => {
    const prev = settingsRef.current!;
    const next = { ...prev, ...patch };
    setSettings(next);
    const result = await SettingsService.Update(next);
    setSettings(result.settings);
    return result;
  };

  const onOpened = (o: SettingsOpened) => {
    if (o.tab >= 0 && o.tab < sections.length) setTab(o.tab);
    setSettings(o.settings);
    if (o.login) setLogin(o.login);
    setOpenCount((n) => n + 1);
  };

  useEffect(() => {
    report(SettingsService.Opened().then(onOpened));
  }, []);
  useEvent<SettingsOpened>("settings:opened", onOpened);
  // The backend polls the server's settings while the Connection tab is
  // showing (and the window is open), and sends the settings when the account
  // or org changes.
  useEvent<Settings>("settings:changed", setSettings);
  useEffect(() => {
    report(SettingsService.SetSettingsActive(tab === connectionTab));
  }, [tab]);
  // Fetch the server's settings whenever the window opens or the Connection
  // tab is selected. The result is returned rather than sent as an event, so it
  // can't be missed while the window is loading or the tab is remounting.
  useEffect(() => {
    if (tab === connectionTab && openCount > 0) report(SettingsService.Refresh().then(setSettings));
  }, [tab, openCount]);

  return (
    <div className="flex h-full min-w-0">
      <nav
        aria-label="Settings sections"
        className="flex w-[192px] shrink-0 flex-col gap-0.5 border-r border-mac-separator bg-mac-sidebar p-2.5 pt-3"
      >
        {sections.map((s, i) => (
          <button
            key={s.name}
            type="button"
            aria-current={i === tab ? "page" : undefined}
            onClick={() => setTab(i)}
            className={cx(
              "flex h-[28px] items-center gap-2 rounded-[6px] px-2 text-left",
              i === tab ? "bg-mac-accent text-white" : "hover:bg-mac-fill",
            )}
          >
            <span className={cx("flex size-[18px] items-center justify-center", i !== tab && "text-mac-accent")}>
              {s.icon}
            </span>
            {s.name}
          </button>
        ))}
      </nav>

      <main className="flex min-w-0 flex-1 flex-col">
        <header className={cx("flex h-[44px] shrink-0 items-center justify-between pr-3", showingLogs ? "pl-2.5" : "pl-5")}>
          <div className="flex min-w-0 items-center gap-1">
            {showingLogs && (
              <button
                type="button"
                aria-label="Back to About"
                onClick={() => setShowingLogs(false)}
                className="flex size-[28px] items-center justify-center rounded-[6px] text-mac-secondary hover:bg-mac-fill"
              >
                <BackIcon />
              </button>
            )}
            <span className="text-[15px] font-semibold">{showingLogs ? "Logs" : sections[tab].name}</span>
          </div>
          <div ref={setHeaderAccessory} className="flex items-center" />
        </header>
        {/* Sections stay mounted so the logs view keeps its scroll position and selection. */}
        <Pane visible={tab === generalTab}>{settings && <GeneralTab settings={settings} apply={apply} />}</Pane>
        <Pane visible={tab === connectionTab}>
          {settings && <ConnectionTab key={openCount} settings={settings} apply={apply} onChange={setSettings} />}
        </Pane>
        <Pane visible={tab === accountsTab}>
          <AccountsTab
            visible={tab === accountsTab}
            openCount={openCount}
            login={login}
            headerAccessory={headerAccessory}
          />
        </Pane>
        <Pane visible={tab === statusTab}>
          <StatusTab key={openCount} />
        </Pane>
        <Pane visible={tab === aboutTab && !showingLogs}>
          <AboutTab onShowLogs={() => setShowingLogs(true)} />
        </Pane>
        <Pane visible={tab === aboutTab && showingLogs}>
          <LogsTab visible={tab === aboutTab && showingLogs} />
        </Pane>
      </main>
    </div>
  );
}

function Pane({ visible, children }: { visible: boolean; children: ReactNode }) {
  return <div className={visible ? "flex min-h-0 min-w-0 flex-1 flex-col" : "hidden"}>{children}</div>;
}

// Sidebar glyphs standing in for the SF Symbols the macOS app uses.

function GearIcon() {
  return (
    <svg viewBox="0 0 20 20" className="size-[16px]" fill="currentColor">
      <path
        fillRule="evenodd"
        d="M11.1 1.5a1 1 0 0 1 1 .8l.3 1.6c.5.2 1 .5 1.4.8l1.5-.6a1 1 0 0 1 1.2.4l1.1 1.9a1 1 0 0 1-.2 1.3l-1.3 1c.1.5.1 1.1 0 1.6l1.3 1a1 1 0 0 1 .2 1.3l-1.1 1.9a1 1 0 0 1-1.2.4l-1.5-.6c-.4.3-.9.6-1.4.8l-.3 1.6a1 1 0 0 1-1 .8H8.9a1 1 0 0 1-1-.8l-.3-1.6c-.5-.2-1-.5-1.4-.8l-1.5.6a1 1 0 0 1-1.2-.4l-1.1-1.9a1 1 0 0 1 .2-1.3l1.3-1a5.6 5.6 0 0 1 0-1.6l-1.3-1a1 1 0 0 1-.2-1.3l1.1-1.9a1 1 0 0 1 1.2-.4l1.5.6c.4-.3.9-.6 1.4-.8l.3-1.6a1 1 0 0 1 1-.8h2.2ZM10 7a3 3 0 1 0 0 6 3 3 0 0 0 0-6Z"
      />
    </svg>
  );
}

/** `network`: the Connection tab. */
function NetworkIcon() {
  return (
    <svg viewBox="0 0 20 20" className="size-[16px]" fill="none" stroke="currentColor" strokeWidth="1.6">
      <circle cx="10" cy="10" r="7.5" />
      <path d="M2.5 10h15M10 2.5c2.1 2.1 3 4.6 3 7.5s-.9 5.4-3 7.5M10 2.5C7.9 4.6 7 7.1 7 10s.9 5.4 3 7.5" />
    </svg>
  );
}

/** `person.crop.circle.fill`. */
function AccountsIcon() {
  return (
    <svg viewBox="0 0 20 20" className="size-[16px]" fill="currentColor">
      <path
        fillRule="evenodd"
        d="M10 1.5a8.5 8.5 0 1 1 0 17 8.5 8.5 0 0 1 0-17Zm0 3.4a2.9 2.9 0 1 0 0 5.8 2.9 2.9 0 0 0 0-5.8ZM5 15.3a6.5 6.5 0 0 0 10 0c-.8-1.6-2.8-2.6-5-2.6s-4.2 1-5 2.6Z"
      />
    </svg>
  );
}

function StatusIcon() {
  return (
    <svg viewBox="0 0 20 20" className="size-[16px]" fill="currentColor">
      <rect x="5" y="1.5" width="10" height="6.5" rx="2" />
      <rect x="5" y="12" width="10" height="6.5" rx="2" />
      <rect x="9.25" y="8" width="1.5" height="4" />
    </svg>
  );
}

/** `chevron.left`, the back button of a pushed page. */
function BackIcon() {
  return (
    <svg viewBox="0 0 10 16" className="h-[14px] w-[9px]" fill="none" stroke="currentColor" strokeWidth="2">
      <path d="M8 2 2 8l6 6" strokeLinecap="round" strokeLinejoin="round" />
    </svg>
  );
}

function InfoIcon() {
  return (
    <svg viewBox="0 0 20 20" className="size-[16px]" fill="currentColor">
      <path
        fillRule="evenodd"
        d="M10 1.5a8.5 8.5 0 1 1 0 17 8.5 8.5 0 0 1 0-17ZM9 9h2v5.5H9V9Zm1-4a1.25 1.25 0 1 1 0 2.5A1.25 1.25 0 0 1 10 5Z"
      />
    </svg>
  );
}
