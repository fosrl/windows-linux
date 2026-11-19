import { useEffect, useState } from "react";
import { AppService, type AppInfo } from "@bindings";
import { Chevron, Form, LinkRow, Row, Section, Value } from "../components/controls";
import { report, urls } from "../lib";

/** `onShowLogs` opens the Logs page, which replaces this one until Back. */
export function AboutTab({ onShowLogs }: { onShowLogs: () => void }) {
  const [info, setInfo] = useState<AppInfo>({ version: "", year: new Date().getFullYear() });
  useEffect(() => {
    report(AppService.Info().then(setInfo));
  }, []);

  return (
    <Form>
      <Section header="Application">
        <Row title="Version">
          <Value>{info.version}</Value>
        </Row>
        <Row title="Copyright">
          <Value>© {info.year} Fossorial, Inc.</Value>
        </Row>
      </Section>

      <Section header="Resources">
        <LinkRow href={urls.docs}>Documentation</LinkRow>
        <LinkRow href={urls.howItWorks}>How Pangolin Works</LinkRow>
      </Section>

      <Section header="Legal">
        <LinkRow href={urls.terms}>Terms of Service</LinkRow>
        <LinkRow href={urls.privacy}>Privacy Policy</LinkRow>
      </Section>

      <Section header="Troubleshooting">
        <Row title="Logs" onClick={onShowLogs}>
          <Chevron />
        </Row>
      </Section>
      <p className="-mt-3.5 px-2.5 text-[11px] leading-[14px] text-mac-secondary">
        Messages from the app and the tunnel. Save them when reporting a problem.
      </p>
    </Form>
  );
}
