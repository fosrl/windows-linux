import type { ReactNode } from "react";

// The banners at the top of the General and Connection tabs, as in the macOS
// app: an icon on the title's line, the detail underneath at full width, and
// an optional action on the right like the settings below.

/** A banner's title and detail. */
export function BannerText({ icon, title, detail }: { icon: ReactNode; title: string; detail: string }) {
  return (
    <div className="flex min-w-0 flex-1 flex-col gap-1.5 text-[13px]">
      <div className="flex items-center gap-1.5 font-semibold break-words">
        <span aria-hidden className="flex size-[15px] shrink-0 items-center justify-center text-mac-secondary">
          {icon}
        </span>
        <span className="min-w-0">{title}</span>
      </div>
      <div className="break-words text-mac-secondary">{detail}</div>
    </div>
  );
}

/** A banner row: the text, with an optional action on the right. */
export function BannerRow({ children, action }: { children: ReactNode; action?: ReactNode }) {
  return (
    <div className="flex items-center gap-4 px-2.5 py-2.5">
      {children}
      {action && <div className="flex shrink-0 items-center">{action}</div>}
    </div>
  );
}

/**
 * A setting's name, with a lock after it when the org's administrator manages
 * it. The banner above the settings explains what the lock means; hovering
 * the name explains it too.
 */
export function SettingTitle({ children, locked, orgName }: { children: string; locked: boolean; orgName: string }) {
  if (!locked) return <>{children}</>;
  return (
    <span
      className="inline-flex items-center gap-1"
      title={`Your administrator in ${orgName} set this for your role. To change it, contact your administrator.`}
    >
      {children}
      <span className="text-mac-secondary" role="img" aria-label="Managed by your administrator">
        <LockIcon />
      </span>
    </span>
  );
}

// Glyphs standing in for the SF Symbols the macOS app uses.

/** `lock.fill`. */
export function LockIcon() {
  return (
    <svg viewBox="0 0 12 14" className="h-[11px] w-[10px]" fill="currentColor">
      <path
        fillRule="evenodd"
        d="M6 1a3 3 0 0 0-3 3v2H2.5A1.5 1.5 0 0 0 1 7.5v4A1.5 1.5 0 0 0 2.5 13h7a1.5 1.5 0 0 0 1.5-1.5v-4A1.5 1.5 0 0 0 9.5 6H9V4a3 3 0 0 0-3-3Zm1.6 5V4a1.6 1.6 0 0 0-3.2 0v2h3.2Z"
      />
    </svg>
  );
}

/** `globe`: the org uses the settings for all organizations. */
export function GlobeIcon() {
  return (
    <svg viewBox="0 0 16 16" className="size-[14px]" fill="none" stroke="currentColor" strokeWidth="1.3">
      <circle cx="8" cy="8" r="6.5" />
      <path d="M1.5 8h13M8 1.5c1.8 1.8 2.6 4 2.6 6.5S9.8 12.7 8 14.5M8 1.5C6.2 3.3 5.4 5.5 5.4 8s.8 4.7 2.6 6.5" />
    </svg>
  );
}

/** `building.2`: the org has its own settings. */
export function BuildingIcon() {
  return (
    <svg viewBox="0 0 16 16" className="size-[14px]" fill="currentColor">
      <path
        fillRule="evenodd"
        d="M2.5 1.5A1 1 0 0 0 1.5 2.5v12h7v-12a1 1 0 0 0-1-1h-5Zm1 2h1.5V5H3.5V3.5Zm3 0H5.5V5H7V3.5Zm-3 3h1.5V8H3.5V6.5Zm3 0H5.5V8H7V6.5Zm-3 3h1.5V11H3.5V9.5Zm3 0H5.5V11H7V9.5ZM9.5 6h4a1 1 0 0 1 1 1v7.5h-5V6Zm1.5 2v1.5h2V8h-2Zm0 3v1.5h2V11h-2Z"
      />
    </svg>
  );
}
