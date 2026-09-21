import React from 'react';
import clsx from 'clsx';
import Link from '@docusaurus/Link';
import useDocusaurusContext from '@docusaurus/useDocusaurusContext';
import Layout from '@theme/Layout';

import styles from './index.module.css';

function HomepageHeader() {
  const {siteConfig} = useDocusaurusContext();
  return (
    <header className={clsx('hero hero--primary', styles.heroBanner)}>
      <div className="container">
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
      title={`Hello from ${siteConfig.title}`}
      description="Linux MCP Daemon Documentation">
      <HomepageHeader />
      <main>
        <div className="container" style={{padding: '2rem 0', textAlign: 'center'}}>
          <h2>Secure, high-performance filesystem management for AI.</h2>
          <p>Built with Go, utilizing Ephemeral Workers for complete UID isolation and Singleflight Caching for performance.</p>
        </div>
      </main>
    </Layout>
  );
}
