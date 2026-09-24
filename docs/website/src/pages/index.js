import React from 'react';
import clsx from 'clsx';
import Link from '@docusaurus/Link';
import useDocusaurusContext from '@docusaurus/useDocusaurusContext';
import useBaseUrl from '@docusaurus/useBaseUrl';
import Layout from '@theme/Layout';

import styles from './index.module.css';

function HomepageHeader() {
  const {siteConfig} = useDocusaurusContext();
  return (
    <header className={clsx('hero hero--primary', styles.heroBanner)}>
      <div className="container">
        <img src={useBaseUrl('/img/mcpd-badge.png')} alt="mcpd: a penguin in sunglasses" width="260" height="260" />
        <h1 className="hero__title">{siteConfig.title}</h1>
        <p className="hero__subtitle">{siteConfig.tagline}</p>
        <div className={styles.buttons}>
          <Link
            className="button button--secondary button--lg"
            to="/intro">
            Read the Documentation ⏱️
          </Link>
        </div>
      </div>
    </header>
  );
}

export default function Home() {
  const {siteConfig} = useDocusaurusContext();
  return (
    <Layout
      title={siteConfig.title}
      description="Linux MCPd - a zero-dependency Go daemon bridging AI agents to a Linux host via MCP">
      <HomepageHeader />
      <main>
        <div className="container" style={{padding: '2rem 0', textAlign: 'center'}}>
          <h2>Kernel-first Linux introspection and administration for AI agents.</h2>
          <p>Files, disks, processes, network, devices, kernel, logs, services, users - reachable over MCP (HTTP/SSE + JSON-RPC) or the schema-discovered <code>linuxctl</code> CLI. Ephemeral, per-call workers under real OS UIDs enforce privilege isolation; nothing runs as root by default.</p>
        </div>
      </main>
    </Layout>
  );
}
