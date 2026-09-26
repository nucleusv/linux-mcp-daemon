import React, {useEffect, useState} from 'react';
import clsx from 'clsx';
import useDocusaurusContext from '@docusaurus/useDocusaurusContext';
import {useLocation} from '@docusaurus/router';

// Every published version is its own static site on GitHub Pages: the latest
// release at the site root, each release under /vX.Y.Z/, main under /next/.
// The list of versions lives in versions.json at the site root - outside all
// of them - and is read here at runtime, so a site published once lists the
// versions released after it without ever being rebuilt.
//
// versions.json: {"latest": "v0.3.3", "versions": ["v0.3.3", ...], "next": true}

function useVersionIndex(siteRoot) {
  const [index, setIndex] = useState(null);
  useEffect(() => {
    fetch(`${siteRoot}versions.json`, {cache: 'no-cache'})
      .then((r) => (r.ok ? r.json() : null))
      .then(setIndex)
      .catch(() => setIndex(null));
  }, [siteRoot]);
  return index;
}

export default function VersionSwitcher({mobile}) {
  const {siteConfig} = useDocusaurusContext();
  const {siteRoot, docsVersion} = siteConfig.customFields;
  const {pathname} = useLocation();
  const index = useVersionIndex(siteRoot);

  const latest = index?.latest;
  // The current release first, then older ones (newest first), main last.
  const releases = index?.versions ?? [];
  const versions = [
    ...releases.filter((v) => v === latest),
    ...releases.filter((v) => v !== latest),
    ...(index?.next ? ['next'] : []),
  ];
  if (!versions.includes(docsVersion)) {
    versions.unshift(docsVersion); // a local build, or a list not fetched yet
  }
  // The same page in the other version (a page it lacks shows the 404 page).
  const page = pathname.startsWith(siteConfig.baseUrl)
    ? pathname.slice(siteConfig.baseUrl.length)
    : '';
  const href = (v) => siteRoot + (v === latest ? '' : `${v}/`) + page;
  const label = (v) =>
    v === 'next' ? 'next (main)' : v === latest ? `current (${v})` : v;

  const links = versions.map((v) => (
    <li key={v}>
      <a
        href={href(v)}
        className={clsx(
          mobile ? 'menu__link' : 'dropdown__link',
          v === docsVersion &&
            (mobile ? 'menu__link--active' : 'dropdown__link--active'),
        )}>
        {label(v)}
      </a>
    </li>
  ));

  if (mobile) {
    return (
      <li className="menu__list-item">
        <span className="menu__link">Version: {label(docsVersion)}</span>
        <ul className="menu__list">{links}</ul>
      </li>
    );
  }
  return (
    <div className="navbar__item dropdown dropdown--hoverable dropdown--right">
      <a
        href="#"
        className="navbar__link"
        role="button"
        aria-haspopup="true"
        aria-label="Documentation version"
        onClick={(e) => e.preventDefault()}>
        {label(docsVersion)}
      </a>
      <ul className="dropdown__menu">{links}</ul>
    </div>
  );
}
